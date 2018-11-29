package buildkite

import (
	"time"

	"fmt"

	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Node struct {
	State int64
	Name  string
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
}

func NewNodemaster(config string) *Nodemaster {
	nm := Nodemaster{config: ReadConfig(config), Nodes: make(map[string]*Node), Builds: make(map[string]*BuildState)}
	nm.done = nm.startChecker()
	return &nm
}

func (nm *Nodemaster) Close() {
	nm.done <- true
	close(nm.done)
}
func (nm *Nodemaster) UpdateNodes() error {
	sess, err := nm.config.AswSession(nm.config.Ec2Region)
	if err != nil {
		return err
	}

	svc := ec2.New(sess)
	filter := make([]*ec2.Filter, 0)
	for _, nt := range nm.config.NodeTags {
		filter = append(filter, &ec2.Filter{Name: aws.String(fmt.Sprintf("tag:%s", nt.Name)),
			Values: []*string{aws.String(nt.Value)}})
	}
	o, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: filter})
	for _, r := range o.Reservations {
		for _, i := range r.Instances {
			name := "n/a"
			for _, tag := range i.Tags {
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
			nm.Nodes[*i.InstanceId] = &Node{State: *(i.State.Code), Name: name}
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
	fmt.Printf("Required nodes %d running %d\n", reqNodes, nm.RunningNodes())
}

func (nm *Nodemaster) startChecker() chan bool {
	done := make(chan bool)

	go func() {
		ticker := time.NewTicker(time.Second * 5)
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
