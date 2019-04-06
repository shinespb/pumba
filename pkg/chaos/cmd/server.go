package cmd

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alexei-led/pumba/pkg/chaos/docker/controller"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type serverContext struct {
	context context.Context
	version string
}

// NewServerCommand initialize CLI server command and bind it to the serverContext
func NewServerCommand(ctx context.Context, version string) *cli.Command {
	cmdContext := &serverContext{context: ctx, version: version}
	return &cli.Command{
		Name: "server",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port, p",
				Usage:  "server port",
				Value:  8080,
				EnvVar: "PORT",
			},
		},
		Usage:       "run in a headless server mode",
		Description: "pumba server provides a REST API for running chaos commands",
		Action:      cmdContext.run,
	}
}

// Server Command
func (cmd *serverContext) run(c *cli.Context) error {
	// get dry-run mode
	// dryRun := c.GlobalBool("dry-run")
	// get server listener port
	port := c.Int("port")

	// configure server
	r := gin.Default()
	// report logs to stdout
	r.Use(gin.Logger())
	// Recovery middleware recovers from any panics and writes a 500 if there was one
	r.Use(gin.Recovery())

	// handle chaos commands
	dockerChaos := controller.NewDockerChaosController(cmd.context)
	r.POST("/docker/kill", dockerChaos.Kill)
	r.POST("/docker/pause", dockerChaos.Pause)
	r.POST("/docker/stop", dockerChaos.Stop)
	r.POST("/docker/remove", dockerChaos.Remove)

	// handle helper commands
	r.GET("/version", cmd.getVersion)

	// create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: r,
	}

	// run server in goroutine
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// wait for server to stop (with Ctrl+C)
	select {
	case <-cmd.context.Done():
		log.Debug("Gracefully shutting down server ...")
	}

	// gracefully shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server shutdown")
		return err
	}
	log.Debug("Server shutdown completed")
	return nil
}

func (cmd *serverContext) getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": cmd.version})
	return
}
