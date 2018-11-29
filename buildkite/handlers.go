package buildkite

import "log"

type pingHandler struct {
}

func (ph pingHandler) Handle(e Event) int {
	log.Printf("Ping recieved from %s\n", e.Sender.Name)
	return 200
}
