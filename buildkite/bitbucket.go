package buildkite

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/gin-gonic/gin"
)

func (bk BuildKite) BitbucketHook(c *gin.Context) {

	c.Request.ParseForm()
	cb := buildkite.CreateBuild{Message: "API"}

	var b Body
	err := json.NewDecoder(c.Request.Body).Decode(&b)
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
	_, _, err = bk.bkCLient.Builds.Create(c.Param("org"), c.Param("pipeline"), &cb)

	if err != nil {
		c.Error(err)
		c.Status(500)
		return
	}
}
