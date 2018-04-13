// Package ginbunyan provides log handling using bunyan package.
// Code structure based on zip package.
package ginbunyan

import (
	"time"

	"github.com/bhoriuchi/go-bunyan/bunyan"
	"github.com/gin-gonic/gin"
)

// Ginbunyan returns a gin.HandlerFunc (middleware) that logs requests using bhoriuchi/go-bunyan.
func Ginbunyan(logger *bunyan.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			for _, e := range c.Errors.Errors() {
				logger.Error(e)
			}
		} else {
			le := make(map[string]interface{})

			le["status"] = c.Writer.Status()
			le["method"] = c.Request.Method
			le["path"] = path
			le["query"] = query
			le["ip"] = c.ClientIP()
			le["userAgent"] = c.Request.UserAgent()
			le["time"] = end.Format(time.RFC3339)
			le["latency"] = latency

			logger.Info(le)
		}
	}
}
