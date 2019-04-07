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

// Pause handler
func (cmd *serverContext) Pause(c *gin.Context) {
	// pause command message
	var msg docker.PauseMessage
	err := c.ShouldBindJSON(&msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// init pause command
	pauseCommand, err := docker.NewPauseCommand(chaos.DockerClient, msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// store chaos job cancel() in jobs map
	context, cancel := context.WithCancel(cmd.context)
	job := fmt.Sprintf("pause-%v", time.Now().UnixNano())
	ctrl.ChaosJobs.Store(job, cancel)
	// run pause command in goroutine
	go chaos.RunChaosCommand(context, pauseCommand, msg.Interval, msg.Random)
	c.JSON(http.StatusAccepted, gin.H{"status": "running pause command ..."})
	return
}
