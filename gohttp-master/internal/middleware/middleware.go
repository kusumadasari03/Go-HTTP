package middleware

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/aakifnehal/gohttp/internal/http"
)

// Middleware represents a middleware function
type Middleware func(HandlerFunc) HandlerFunc

// HandlerFunc represents a handler function
type HandlerFunc func(req *http.Request, conn net.Conn) *http.Response

// RequestStats tracks server statistics
type RequestStats struct {
	TotalRequests  int64
	ActiveRequests int64
	TotalResponses int64
	ErrorResponses int64
}

var Stats RequestStats

// LoggingMiddleware logs requests and responses
func LoggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *http.Request, conn net.Conn) *http.Response {
		start := time.Now()

		// Increment counters
		atomic.AddInt64(&Stats.TotalRequests, 1)
		atomic.AddInt64(&Stats.ActiveRequests, 1)

		// Get client address
		clientAddr := conn.RemoteAddr().String()

		// Log request
		fmt.Printf("[%s] %s %s %s - %s\n",
			start.Format("15:04:05"),
			req.Method,
			req.Path,
			req.Version,
			clientAddr)

		// Call next handler
		response := next(req, conn)

		// Calculate duration
		duration := time.Since(start)

		// Update counters
		atomic.AddInt64(&Stats.ActiveRequests, -1)
		atomic.AddInt64(&Stats.TotalResponses, 1)

		if response.StatusCode >= 400 {
			atomic.AddInt64(&Stats.ErrorResponses, 1)
		}

		// Log response
		fmt.Printf("[%s] %d %s - %v - %s\n",
			time.Now().Format("15:04:05"),
			response.StatusCode,
			response.StatusText,
			duration,
			clientAddr)

		return response
	}
}

// RateLimitMiddleware provides basic rate limiting per connection
func RateLimitMiddleware(maxConcurrent int64) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *http.Request, conn net.Conn) *http.Response {
			current := atomic.LoadInt64(&Stats.ActiveRequests)

			if current > maxConcurrent {
				log.Printf("Rate limit exceeded: %d active requests (max: %d)", current, maxConcurrent)
				return http.NewResponse(503, "Service Unavailable")
			}

			return next(req, conn)
		}
	}
}

// RecoveryMiddleware recovers from panics in handlers
func RecoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *http.Request, conn net.Conn) *http.Response {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v", r)
				// Return 500 error response
				errorResp := http.NewInternalErrorResponse()
				errorResp.WriteResponse(conn)
			}
		}()

		return next(req, conn)
	}
}

// StatsMiddleware provides a /stats endpoint
func GetStatsResponse() *http.Response {
	total := atomic.LoadInt64(&Stats.TotalRequests)
	active := atomic.LoadInt64(&Stats.ActiveRequests)
	responses := atomic.LoadInt64(&Stats.TotalResponses)
	errors := atomic.LoadInt64(&Stats.ErrorResponses)

	var errorRate float64
	if responses > 0 {
		errorRate = float64(errors) / float64(responses) * 100
	}

	statsJSON := fmt.Sprintf(`{
  "server": "Custom HTTP Server",
  "version": "1.0.0",
  "stats": {
    "total_requests": %d,
    "active_requests": %d,
    "total_responses": %d,
    "error_responses": %d,
    "error_rate_percent": %.2f
  },
  "timestamp": "%s"
}`, total, active, responses, errors, errorRate, time.Now().Format(time.RFC3339))

	return http.NewJSONResponse(statsJSON)
}

// Chain combines multiple middlewares
func Chain(middlewares ...Middleware) Middleware {
	return func(final HandlerFunc) HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
