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

// Remove handler
func (cmd *serverContext) Remove(c *gin.Context) {
	// get message
	var msg docker.RemoveMessage
	err := c.ShouldBindJSON(&msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// init remove command
	removeCommand, err := docker.NewRemoveCommand(chaos.DockerClient, msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// store chaos job cancel() in jobs map
	context, cancel := context.WithCancel(cmd.context)
	job := fmt.Sprintf("remove-%v", time.Now().UnixNano())
	ctrl.ChaosJobs.Store(job, cancel)
	// run remove command in goroutine
	go chaos.RunChaosCommand(context, removeCommand, msg.Interval, msg.Random)
	c.JSON(http.StatusAccepted, gin.H{"status": "running remove command..."})
	return
}
