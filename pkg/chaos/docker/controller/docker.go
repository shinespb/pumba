package controller

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
)

// keep all chaos jobs in sync.Map
var jobs sync.Map

type serverContext struct {
	context context.Context
}

// DockerChaos controller interface
type DockerChaos interface {
	CancelChaos(c *gin.Context)
	Kill(c *gin.Context)
	Pause(c *gin.Context)
	Remove(c *gin.Context)
	Stop(c *gin.Context)
}

// NewDockerChaosController dockerContext
func NewDockerChaosController(ctx context.Context) DockerChaos {
	return &serverContext{context: ctx}
}
