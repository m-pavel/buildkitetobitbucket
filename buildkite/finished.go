package buildkite

import "log"

type finishedHandler struct {
	Notify *Notify
	nm     *Nodemaster
}

func newFinishedHandler(config string, nm *Nodemaster) *finishedHandler {
	fh := finishedHandler{nm: nm}
	fh.Notify = NewNotify(config)
	return &fh
}

func (fh finishedHandler) Handle(e Event) int {
	switch e.Build.State {
	case "passed":
		break
	case "failed":
		err := fh.Notify.SendFail(e)
		if err != nil {
			log.Println(err)
		}
	}
	fh.nm.StopBuild(e)
	return 200
}
