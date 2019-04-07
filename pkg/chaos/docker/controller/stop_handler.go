package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alexei-led/pumba/pkg/chaos"
	ctrl "github.com/alexei-led/pumba/pkg/chaos/controller"
	"github.com/alexei-led/pumba/pkg/chaos/docker"

	"github.com/gin-gonic/gin"
)

// Stop handler
func (cmd *serverContext) Stop(c *gin.Context) {
	// get stop message
	var msg docker.StopMessage
	err := c.ShouldBindJSON(&msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// init stop command
	stopCommand, err := docker.NewStopCommand(chaos.DockerClient, msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// store chaos job cancel() in jobs map
	context, cancel := context.WithCancel(cmd.context)
	job := fmt.Sprintf("stop-%v", time.Now().UnixNano())
	ctrl.ChaosJobs.Store(job, cancel)
	// run stop command in goroutine
	go chaos.RunChaosCommand(context, stopCommand, msg.Interval, msg.Random)
	c.JSON(http.StatusAccepted, gin.H{"status": "running stop command ..."})
	return
}
