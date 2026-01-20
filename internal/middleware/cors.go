package middleware

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowed string) gin.HandlerFunc {
	origins := map[string]struct{}{}
	allowAll := allowed == "*"

	log.Println("CORS origins:", allowed)

	if !allowAll {
		for _, o := range strings.Split(allowed, ",") {
			origins[strings.TrimSpace(o)] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if _, ok := origins[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Timezone")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
