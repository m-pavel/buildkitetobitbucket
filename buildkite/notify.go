package buildkite

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/go-errors/errors"
)

const (
	// TODO use text/template
	subjectFail  = "Buildkite build failed %s"
	bodyFailText = "Buildkite build %d failed\n%s"
	bodyFailHtml = "Buildkite build %d failed<br>%s"
	charSet      = "UTF-8"
)

type Notify struct {
	Config NotifyConfig
}

func NewNotify(config string) *Notify {
	n := Notify{Config: *ReadConfig(config)}

	return &n
}
func (n *Notify) SendFail(event Event) error {
	return n.sendEmail(event, subjectFail, bodyFailHtml, bodyFailText)
}

func (n *Notify) sendEmail(event Event, subj, htmlbody, textbody string) error {
	if !n.Config.NotifyEnabled {
		return errors.New("Notification disabled")
	}
	ppl := n.Config.Pipeline(event.Pipeline.Slug)
	if ppl == nil {
		return errors.New(fmt.Sprintf("Unconfigured pipeline %s", event.Pipeline.Slug))
	}

	if ppl.Notify == nil || len(ppl.Notify) == 0 {
		return errors.New("Empty notification list")
	}

	sess, err := n.Config.AswSession(n.Config.NotifyRegion)
	if err != nil {
		return err
	}
	svc := ses.New(sess)

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: ppl.Adresses(),
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(fmt.Sprintf(htmlbody, event.Build.Number, event.Build.Message)),
				},
				Text: &ses.Content{
					Charset: aws.String(charSet),
					Data:    aws.String(fmt.Sprintf(textbody, event.Build.Number, event.Build.Message)),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charSet),
				Data:    aws.String(fmt.Sprintf(subj, event.Pipeline.Name)),
			},
		},
		Source: aws.String(n.Config.Sender),
	}

	_, err = svc.SendEmail(input)
	return err

}
