package buildkite

import (
	"os"

	"log"

	"github.com/gin-gonic/gin"
)

const (
	buildkiteApiToken = "buildkiteApiToken"
)

type BuildKite struct {
	router *Router
	nm     *Nodemaster
}

func NewBuildKite(config string) *BuildKite {
	bk := BuildKite{}
	bk.nm = NewNodemaster(config)
	bk.router = NewRouter(bk.nm, config)
	return &bk
}

func (bk BuildKite) BuildkiteHook(c *gin.Context) {
	token := os.Getenv(buildkiteApiToken)
	if token == "" {
		log.Println("WARNING: No buildkite token specified. Unable to check security")
	} else {
		if token != c.Request.Header.Get("X-Buildkite-Token") {
			log.Println("ERROR: Wrong request token")
			c.Status(401)
			return
		}
	}

	bk.router.Route(c)
}
