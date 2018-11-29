package buildkite

import (
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/ghodss/yaml"
)

type PipelineConfig struct {
	Name   string
	Nodes  int
	Notify []string
}

type Nodetag struct {
	Name  string
	Value string
}

type NotifyConfig struct {
	NotifyRegion  string `yaml:"notifyregion"`
	Ec2Region     string `yaml:"ec2region"`
	Sender        string
	NodeTags      []Nodetag
	AwsProfile    string `yaml:"awsprofile"`
	Pipelines     []PipelineConfig
	NotifyEnabled bool `yaml:"notifyenabled"`
}

func ReadConfig(path string) *NotifyConfig {
	n := NotifyConfig{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println(err)
		return &n
	}

	err = yaml.Unmarshal(yamlFile, &n)
	if err != nil {
		n.NotifyEnabled = false
		log.Println(err)
		return &n
	}
	for i := range n.Pipelines {
		if n.Pipelines[i].Nodes == 0 {
			n.Pipelines[i].Nodes = 1
		}
	}
	return &n
}

func (nc NotifyConfig) Pipeline(id string) *PipelineConfig {
	for _, p := range nc.Pipelines {
		if p.Name == id {
			return &p
		}
	}
	return nil
}

func (p PipelineConfig) Adresses() []*string {
	res := make([]*string, len(p.Notify))
	for i := 0; i < len(res); i++ {
		res[i] = aws.String(p.Notify[i])
	}
	return res
}

func (nc NotifyConfig) AswSession(region string) (*session.Session, error) {
	awsCred := credentials.NewEnvCredentials()
	if nc.AwsProfile != "" {
		awsCred = credentials.NewSharedCredentials("", nc.AwsProfile)
	}
	return session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: awsCred},
	)
}
