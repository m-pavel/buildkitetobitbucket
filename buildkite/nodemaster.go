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

func (n Node) Start() {
	log.Printf("Starting node %s\n", n.ID)
}

func (n Node) Stop() {
	log.Printf("Stoping node %s\n", n.ID)
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
}

func NewNodemaster(config string, client *buildkite.Client) (*Nodemaster, error) {
	nm := Nodemaster{config: ReadConfig(config), Nodes: make(map[string]*Node), Builds: make(map[string]*BuildState)}
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
	nm.UpdateNodes()

	if nm.RunningNodes() > reqNodes {
		nm.ShutdownNodes(nm.RunningNodes() - reqNodes)
	}
	if reqNodes > nm.RunningNodes() {
		nm.StartNodes(reqNodes - nm.RunningNodes())
	}
	log.Printf("Required nodes %d running %d, total %d\n", reqNodes, nm.RunningNodes(), nm.TotalNodes())
}
func (nm *Nodemaster) ShutdownNodes(num int) {
	for i := 0; i < num; i++ {
		node := nm.GetRandomNode(NodeRunning)
		if node.CanShutDown(nm.client) {
			log.Printf("Shutting down node %s\n", node.ID)
		}
	}
}

func (nm *Nodemaster) StartNodes(num int) {
	for i := 0; i < num; i++ {
		node := nm.GetRandomNode(NodeStopped)
		log.Printf("Starting node %s\n", node.ID)
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
		ticker := time.NewTicker(time.Second * 5) // TODO Minutes
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
