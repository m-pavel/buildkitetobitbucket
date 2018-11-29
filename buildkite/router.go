package buildkite

import (
	"github.com/gin-gonic/gin"
)

type EventHandler interface {
	Handle(Event) int
}

type Router struct {
	handlers map[string]EventHandler
}

func NewRouter(nm *Nodemaster, cfg string) *Router {
	r := Router{handlers: make(map[string]EventHandler)}
	r.handlers[EventPing] = &pingHandler{}
	r.handlers[EventBuildRunning] = newStartedHandler(cfg, nm)
	r.handlers[EventBuildFinished] = newFinishedHandler(cfg, nm)
	return &r
}

func (r Router) Route(c *gin.Context) {
	header := c.Request.Header.Get("X-Buildkite-Event")
	handler := r.handlers[header]
	if handler == nil {
		c.String(400, "No header")
		return
	}
	var e Event
	err := c.BindJSON(&e)
	if err != nil {
		c.String(500, "%v", err)
		return
	}
	c.Status(handler.Handle(e))
}
