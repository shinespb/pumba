package controller

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// ChaosJobs - keep all chaos jobs in sync.Map
var ChaosJobs sync.Map

// ChaosController controller interface
type ChaosController interface {
	Cancel(c *gin.Context)
	GetVersion(c *gin.Context)
}

type chaosController struct {
	version string
}

// NewChaosController generic controller handlers
func NewChaosController(version string) ChaosController {
	return &chaosController{version}
}
