package buildkite

import (
	"time"

	"fmt"

	"log"

	"math/rand"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/buildkite/go-buildkite/buildkite"
)

const (
	NodeRunning = 16
	NodeStopped = 80
)

type Node struct {
	ID    string
	State int64
	Name  string
}

func (n Node) Start(ec2p *ec2.EC2) error {
	log.Printf("Starting node %s\n", n.ID)
	_, err := ec2p.StartInstances(&ec2.StartInstancesInput{InstanceIds: []*string{aws.String(n.ID)}})
	return err
}

func (n Node) Stop(ec2p *ec2.EC2) error {
	log.Printf("Stoping node %s\n", n.ID)
	_, err := ec2p.StopInstances(&ec2.StopInstancesInput{InstanceIds: []*string{aws.String(n.ID)}})
	return err
}

func (n Node) CanShutDown(c *buildkite.Client) bool {
	if n.State != 16 {
		return false // does not makes sense
	}

	a, _, err := c.Agents.List("autogrow-systems-limited", nil)
	if err != nil {
		log.Println(err)
		return false
	}

	nodetag := fmt.Sprintf("aws:instance-id=%s", n.ID)
	for _, ag := range a {
		for _, m := range ag.Metadata {
			if m == nodetag {
				if ag.Job == nil {
					return true
				}
			}
		}
	}

	return false
}

type BuildState struct {
	Nodes   int
	Running bool
}
type Nodemaster struct {
	config   *NotifyConfig
	SyncTime *time.Time

	Nodes  map[string]*Node
	Builds map[string]*BuildState

	done chan bool

	client *buildkite.Client
	ec2    *ec2.EC2
	check  time.Duration
}

func NewNodemaster(config string, client *buildkite.Client, ch time.Duration) (*Nodemaster, error) {
	cfg, err := ReadConfig(config)
	if err != nil {
		return nil, err
	}
	nm := Nodemaster{config: cfg, Nodes: make(map[string]*Node), Builds: make(map[string]*BuildState), check: ch}
	nm.done = nm.startChecker()

	nm.client = client

	sess, err := nm.config.AswSession(nm.config.Ec2Region)
	if err != nil {
		return nil, err
	}

	nm.ec2 = ec2.New(sess)

	return &nm, nil
}

func (nm *Nodemaster) Close() {
	nm.done <- true
	close(nm.done)
}
func (nm *Nodemaster) UpdateNodes() error {
	filter := make([]*ec2.Filter, 0)
	for _, nt := range nm.config.NodeTags {
		filter = append(filter, &ec2.Filter{Name: aws.String(fmt.Sprintf("tag:%s", nt.Name)),
			Values: []*string{aws.String(nt.Value)}})
	}
	o, err := nm.ec2.DescribeInstances(&ec2.DescribeInstancesInput{Filters: filter})
	for _, r := range o.Reservations {
		for _, i := range r.Instances {
			name := "n/a"
			for _, tag := range i.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
			nm.Nodes[*i.InstanceId] = &Node{State: *(i.State.Code), Name: name, ID: *i.InstanceId}
		}
	}
	return err
}

func (nm *Nodemaster) RunningNodes() int {
	res := 0
	for _, v := range nm.Nodes {
		if v.State == 16 {
			res++
		}
	}
	return res
}

func (nm Nodemaster) TotalNodes() int {
	return len(nm.Nodes)
}

func (nm *Nodemaster) StartBuild(e Event) {
	b, ok := nm.Builds[e.Pipeline.Slug]
	if !ok {
		nm.Builds[e.Pipeline.Slug] = &BuildState{}
		b = nm.Builds[e.Pipeline.Slug]
	}
	if b.Running {
		log.Printf("WARNING Build %s already running", e.Pipeline.Slug)
	}
	b.Running = true
	b.Nodes = nm.config.Pipeline(e.Pipeline.Slug).Nodes
}

func (nm *Nodemaster) StopBuild(e Event) {
	b, ok := nm.Builds[e.Pipeline.Slug]
	if !ok {
		nm.Builds[e.Pipeline.Slug] = &BuildState{}
		b = nm.Builds[e.Pipeline.Slug]
	}
	if !b.Running {
		log.Printf("WARNING Build %s already stopped", e.Pipeline.Slug)
	}
	b.Running = false
	b.Nodes = nm.config.Pipeline(e.Pipeline.Slug).Nodes
}

func (nm *Nodemaster) Check() {
	reqNodes := 0
	for _, v := range nm.Builds {
		if v.Running {
			reqNodes += v.Nodes
		}
	}
	err := nm.UpdateNodes()
	if err != nil {
		log.Println(err)
		return
	}
	if nm.RunningNodes() > reqNodes && nm.RunningNodes() > nm.config.Warmupnodes {
		nm.ShutdownNodes(nm.RunningNodes() - reqNodes - nm.config.Warmupnodes)
	}

	if reqNodes > nm.RunningNodes() {
		nm.StartNodes(reqNodes - nm.RunningNodes())
	}
	log.Printf("Required nodes %d (%d) running %d, total %d\n", reqNodes, nm.config.Warmupnodes, nm.RunningNodes(), nm.TotalNodes())
}
func (nm *Nodemaster) ShutdownNodes(num int) {
	for i := 0; i < num; i++ {
		node := nm.GetRandomNode(NodeRunning)
		if node == nil {
			log.Println("Unable to find running node")
		} else {
			if node.CanShutDown(nm.client) {
				err := node.Stop(nm.ec2)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func (nm *Nodemaster) StartNodes(num int) {
	for i := 0; i < num; i++ {
		node := nm.GetRandomNode(NodeStopped)
		if node == nil {
			log.Println("Unable to find free node")
		} else {
			err := node.Start(nm.ec2)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (nm *Nodemaster) GetRandomNode(state int) *Node {
	list := make([]*Node, 0)
	for _, n := range nm.Nodes {
		if n.State == int64(state) {
			list = append(list, n)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return list[rand.Intn(len(list))]
}

func (nm *Nodemaster) startChecker() chan bool {
	done := make(chan bool)

	go func() {
		ticker := time.NewTicker(nm.check)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				fmt.Println("Done!")
				return
			case <-ticker.C:
				nm.Check()
			}
		}
	}()
	return done
}
