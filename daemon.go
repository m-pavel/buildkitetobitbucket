package main

import (
	"log"

	"os"

	"flag"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/m-pavel/buildkitetobitbucket/buildkite"
	daemon "github.com/sevlyar/go-daemon"
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

	var logf = flag.String("log", "daemon.log", "log")
	var pid = flag.String("pid", "daemon.pid", "pid")
	var notdaemonize = flag.Bool("n", false, "Do not do to background.")
	var signal = flag.String("s", "", `send signal to the daemon stop — shutdown`)
	var config = flag.String("c", "", `configuration file`)
	flag.Parse()

	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)

	cntxt := &daemon.Context{
		PidFileName: *pid,
		PidFilePerm: 0644,
		LogFileName: *logf,
		LogFilePerm: 0640,
		WorkDir:     "/tmp",
		Umask:       027,
		Args:        os.Args,
	}

	if !*notdaemonize && len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatalf("Unable send signal to the daemon: %v", err)
		}
		daemon.SendCommands(d)
		return
	}

	if os.Getenv(bitbucketApiToken) == "" {
		log.Fatalf("Environment variable '%s' must be specified.", bitbucketApiToken)
	}

	if !*notdaemonize {
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatalln(err)
		}
		if d != nil {
			return
		}
	}

	daemonfunc(*config)
}

func daemonfunc(config string) {
	engine := gin.New()
	bk := buildkite.NewBuildKite(config)

	dgroup := engine.Group("/v1")
	dgroup.GET("/start/:org/:pipeline/:branch", BitbucketHook)
	dgroup.POST("/start/:org/:pipeline/:branch", BitbucketHook)

	dgroup.POST("/buildkite", bk.BuildkiteHook)

	engine.Run(":8080")
}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}
