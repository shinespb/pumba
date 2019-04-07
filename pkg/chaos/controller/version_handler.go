package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (cc *chaosController) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": cc.version})
	return
}
