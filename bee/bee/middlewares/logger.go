package middlewares

import (
	"bee"
	"log"
	"time"
)

func Logger() bee.HandlerFunc {
	return func(c *bee.Context) {
		// Start timer
		t := time.Now()
		// Process request
		c.Next()
		// Calculate resolution time
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
