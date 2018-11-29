package buildkite

import (
	"testing"
	"time"
)

func TestNc(t *testing.T) {
	nc := NewNodemaster("notify.yml")
	defer nc.Close()
	nc.UpdateNodes()
	time.Sleep(time.Minute)
}
