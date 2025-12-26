package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

const TimezoneContextKey = "timezone"

func TimezoneMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tz := c.GetHeader("X-Timezone")

		var loc *time.Location
		if tz != "" {
			parsed, err := time.LoadLocation(tz)
			if err == nil {
				loc = parsed
			}
		}

		if loc == nil {
			loc = time.UTC
		}

		c.Set(TimezoneContextKey, loc)
		c.Next()
	}
}

func GetTimezoneFromContext(c *gin.Context) *time.Location {
	if loc, exists := c.Get(TimezoneContextKey); exists {
		if timezone, ok := loc.(*time.Location); ok {
			return timezone
		}
	}
	return time.UTC
}
