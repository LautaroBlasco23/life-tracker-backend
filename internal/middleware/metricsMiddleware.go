package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	metrics "life-tracker-backend/internal/domain/monitoring"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Observe(duration)

		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Inc()

		if c.Writer.Status() >= 400 {
			metrics.HTTPErrorsTotal.WithLabelValues(
				c.Request.Method,
				path,
			).Inc()
		}
	}
}
