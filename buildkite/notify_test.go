package buildkite

import (
	"testing"
)

func Test001(t *testing.T) {
	n := NewNotify("notify.yml")
	err := n.SendFail(Event{Build: Build{Message: "Test message"}, Pipeline: Pipeline{Name: "Pipeline name", Slug: "testpipeline"}})
	if err != nil {
		t.Fatal(err)
	}
}
