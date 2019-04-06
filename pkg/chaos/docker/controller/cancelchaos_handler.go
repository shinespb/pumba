package controller

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CancelChaos handler
func (cmd *serverContext) CancelChaos(c *gin.Context) {
	// get chaos job
	job := c.Query("job")
	if job == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing chaos job id"})
		return
	}
	// get cancel function
	fn, ok := jobs.Load(job)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "cannot find chaos job with specified id"})
		return
	}
	// cast and call cancel
	cancel, ok := fn.(context.CancelFunc)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected casting error"})
		return
	}
	cancel()

	c.JSON(http.StatusOK, gin.H{"status": "canceled chaos command", "job": job})
	return
}
