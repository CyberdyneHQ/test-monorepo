package main

import (
	"fmt"
	"golang.org/x/text/language"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		tag := language.English
		c.String(200, fmt.Sprintf("Lang: %s", tag))
	})
	r.Run(":8081")
}
