package main

import (
	"log"

	"os"

	"fmt"

	"encoding/json"

	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/gin-gonic/gin"
)

const (
	apiToken = "apiToken"
)

type Repository struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type Actor struct {
	Username    string `json:"username"`
	Displayname string `json:"display_name"`
}
type Body struct {
	Repo  Repository `json:"repository"`
	Actor Actor      `json:"actor"`
}

func main() {

	if os.Getenv(apiToken) == "" {
		log.Fatalf("Environment variable %s must be specified.", apiToken)
	}
	engine := gin.Default()

	dgroup := engine.Group("/v1")
	dgroup.GET("/start/:org/:pipeline/:branch", Hook)
	dgroup.POST("/start/:org/:pipeline/:branch", Hook)

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

	var b Body
	err = json.NewDecoder(c.Request.Body).Decode(&b)
	if err != nil {
		log.Println(err)
	}

	cb.Message = c.Request.Form.Get("message")

	if cb.Message == "" {
		if c.Request.Form.Get("repository") != "" {
			cb.Message = fmt.Sprintf("Started by changes in %s", c.Request.Form.Get("repository"))
		} else {
			cb.Message = fmt.Sprintf("Started by changes in %s by %s", b.Repo.Name, b.Actor.Displayname)
		}
	}

	cb.Branch = c.Param("branch")
	cb.Commit = "HEAD"
	_, _, err = client.Builds.Create(c.Param("org"), c.Param("pipeline"), &cb)

	if err != nil {
		c.Error(err)
		c.Status(500)
		return
	}
}
