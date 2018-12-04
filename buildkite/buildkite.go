package buildkite

import (
	"os"

	"log"

	"time"

	bkt "github.com/buildkite/go-buildkite/buildkite"
	"github.com/gin-gonic/gin"
)

const (
	HookToken = "buildkiteHookToken"
	ApiToken  = "apiToken"
)

type BuildKite struct {
	router   *Router
	nm       *Nodemaster
	bkCLient *bkt.Client
}

func NewBuildKite(config string, timeout int) (*BuildKite, error) {
	bk := BuildKite{}

	bkconfig, err := bkt.NewTokenConfig(os.Getenv(ApiToken), true)
	if err != nil {
		return nil, err
	}

	bk.bkCLient = bkt.NewClient(bkconfig.Client())
	bk.nm, err = NewNodemaster(config, bk.bkCLient, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, err
	}
	bk.router, err = NewRouter(bk.nm, config)
	return &bk, err
}

func (bk BuildKite) BuildkiteHook(c *gin.Context) {
	token := os.Getenv(HookToken)
	if token == "" {
		log.Println("WARNING: No buildkite hook token specified. Unable to check security")
	} else {
		if token != c.Request.Header.Get("X-Buildkite-Token") {
			log.Println("ERROR: Wrong request token")
			c.Status(401)
			return
		}
	}

	bk.router.Route(c)
}
