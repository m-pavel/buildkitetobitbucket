package main

import (
	"log"

	"os"

	"fmt"

	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/gin-gonic/gin"
)

const (
	apiToken = "apiToken"
)

func main() {

	if os.Getenv(apiToken) == "" {
		log.Fatalf("Environment variable %s must be specified.", apiToken)
	}
	engine := gin.Default()

	dgroup := engine.Group("/v1")
	dgroup.GET("/start/:org/:pipeline", Hook)

	engine.Run(":8080")
}

func Hook(c *gin.Context) {
	config, err := buildkite.NewTokenConfig(os.Getenv(apiToken), true)
	if err != nil {
		c.Error(err)
		c.Status(500)
		return
	}

	client := buildkite.NewClient(config.Client())

	c.Request.ParseForm()

	cb := buildkite.CreateBuild{Message: "API"}
	cb.Message = c.Request.Form.Get("message")
	if cb.Message == "" {
		if c.Request.Form.Get("repository") != "" {
			cb.Message = fmt.Sprintf("Started by changes in %s", c.Request.Form.Get("repository"))
		} else {
			cb.Message = "Automatically stated"
		}
	}

	cb.Branch = "master"
	cb.Commit = "HEAD"
	_, _, err = client.Builds.Create(c.Param("org"), c.Param("pipeline"), &cb)

	if err != nil {
		c.Error(err)
		c.Status(500)
		return
	}
}
