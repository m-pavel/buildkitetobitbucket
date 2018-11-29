package buildkite

type startedHandler struct {
	nm *Nodemaster
}

func newStartedHandler(config string, nm *Nodemaster) *startedHandler {
	fh := startedHandler{nm: nm}
	return &fh
}

func (fh startedHandler) Handle(e Event) int {
	fh.nm.StartBuild(e)
	return 200
}
