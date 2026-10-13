package controllers

import "github.com/gin-gonic/gin"

type PBSController struct{}

// Default handles the default PBS endpoint
func (p *PBSController) Default(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "success",
		"message": "PBS endpoint is working",
	})
	c.Abort()
	return
}
