package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aakifnehal/gohttp/internal/config"
	"github.com/aakifnehal/gohttp/internal/http"
	"github.com/aakifnehal/gohttp/internal/middleware"
)

func main() {
	// Parse command line flags
	var configPath = flag.String("config", "config.json", "Path to configuration file")
	var showVersion = flag.Bool("version", false, "Show version information")
	var runTests = flag.Bool("test", false, "Run integration tests against running server")
	flag.Parse()

	if *showVersion {
		fmt.Println("GoHTTP Server v1.0.0")
		fmt.Println("A learning project implementing HTTP server from scratch in Go")
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	if *runTests {
		fmt.Println("Running integration tests...")
		err := runIntegrationTests(cfg.Address())
		if err != nil {
			log.Fatalf("Integration tests failed: %v", err)
		}
		fmt.Println("All integration tests passed!")
		return
	}

	fmt.Printf("Starting HTTP server on %s\n", cfg.Address())
	fmt.Println("Configuration loaded successfully")

	// Listen for TCP connections
	listener, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Printf("Server listening on http://%s\n", cfg.Address())
	fmt.Println("Available endpoints:")
	fmt.Printf("  http://%s/\n", cfg.Address())
	fmt.Printf("  http://%s/hello\n", cfg.Address())
	fmt.Printf("  http://%s/stats\n", cfg.Address())
	fmt.Printf("  http://%s/config\n", cfg.Address())
	fmt.Printf("  curl -X POST http://%s/echo -d 'test data'\n", cfg.Address())

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received, gracefully shutting down...")
		cancel()
		listener.Close()
	}()

	// Store config in global variable for handlers
	serverConfig = cfg

	// Accept connections in a loop
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Waiting for active connections to finish...")
			wg.Wait()
			fmt.Println("Server shut down gracefully")
			return
		default:
			// Use a timeout for accept to allow periodic checking for shutdown
			if tcpListener, ok := listener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			conn, err := listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Check for shutdown signal
				}
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("Failed to accept connection: %v", err)
				}
				continue
			}

			// Handle each connection concurrently
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				handleConnection(c)
			}(conn)
		}
	}
}

var serverConfig *config.Config

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Parse the HTTP request using our custom parser with config
	req, err := http.ParseRequestWithConfig(conn, serverConfig.Limits.MaxRequestBodySize)
	if err != nil {
		log.Printf("   Failed to parse HTTP request: %v", err)
		// Send error response
		errorResp := http.NewBadRequestResponse(err.Error())
		errorResp.WriteResponse(conn)
		return
	}

	// Create the core handler
	coreHandler := func(req *http.Request, conn net.Conn) *http.Response {
		return routeRequest(req)
	}

	// Apply middleware chain
	middlewareChain := middleware.Chain(
		middleware.RecoveryMiddleware,
		middleware.LoggingMiddleware,
		middleware.RateLimitMiddleware(int64(serverConfig.Limits.MaxConcurrentRequests)),
	)

	handler := middlewareChain(coreHandler)

	// Execute handler with middleware
	response := handler(req, conn)

	// Send the response
	err = response.WriteResponse(conn)
	if err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
}

// routeRequest handles the core routing logic
func routeRequest(req *http.Request) *http.Response {
	switch {
	case req.Path == "/" && req.Method == "GET":
		// Serve the main index.html file
		return http.ServeStaticFile("static/index.html")
	case req.Path == "/hello" && req.Method == "GET":
		return http.NewOKResponse("Hello, World! 👋")
	case req.Path == "/info" && req.Method == "GET":
		userAgent := req.Headers["user-agent"]
		if userAgent == "" {
			userAgent = "Unknown"
		}
		body := fmt.Sprintf("Request Info:\nMethod: %s\nPath: %s\nUser-Agent: %s", req.Method, req.Path, userAgent)
		return http.NewOKResponse(body)
	case req.Path == "/echo" && req.Method == "POST":
		// Echo back the request body
		if req.Body == "" {
			return http.NewBadRequestResponse("POST body is required for /echo endpoint")
		} else {
			echoResponse := fmt.Sprintf(`{
  "message": "Echo successful",
  "method": "%s",
  "received_body": "%s",
  "content_length": %d
}`, req.Method, req.Body, len(req.Body))
			return http.NewJSONResponse(echoResponse)
		}
	case req.Path == "/stats" && req.Method == "GET":
		// Return server statistics
		return middleware.GetStatsResponse()
	case req.Path == "/config" && req.Method == "GET":
		// Return current server configuration
		return getConfigResponse()
	case req.Path == "/json" && req.Method == "GET":
		return http.ServeStaticFile("static/api.json")
	case strings.HasPrefix(req.Path, "/static/") && req.Method == "GET":
		// Handle static file requests
		filePath := req.Path[1:] // Remove leading slash
		return http.ServeStaticFile(filePath)
	case req.Method != "GET" && req.Method != "POST":
		return http.NewMethodNotAllowedResponse()
	default:
		return http.NewNotFoundResponse()
	}
}

// getConfigResponse returns the current server configuration
func getConfigResponse() *http.Response {
	if serverConfig == nil {
		return http.NewInternalErrorResponse()
	}

	configJSON := fmt.Sprintf(`{
  "server": {
    "host": "%s",
    "port": %d,
    "address": "%s",
    "read_timeout_seconds": %d,
    "write_timeout_seconds": %d,
    "idle_timeout_seconds": %d
  },
  "limits": {
    "max_concurrent_requests": %d,
    "max_request_body_size_bytes": %d,
    "request_timeout_seconds": %d
  },
  "logging": {
    "enabled": %t,
    "level": "%s",
    "color_output": %t
  },
  "version": "1.0.0"
}`,
		serverConfig.Server.Host,
		serverConfig.Server.Port,
		serverConfig.Address(),
		serverConfig.Server.ReadTimeout,
		serverConfig.Server.WriteTimeout,
		serverConfig.Server.IdleTimeout,
		serverConfig.Limits.MaxConcurrentRequests,
		serverConfig.Limits.MaxRequestBodySize,
		serverConfig.Limits.RequestTimeoutSeconds,
		serverConfig.Logging.Enabled,
		serverConfig.Logging.Level,
		serverConfig.Logging.ColorOutput)

	return http.NewJSONResponse(configJSON)
}

// runIntegrationTests runs basic integration tests against a running server
func runIntegrationTests(serverAddr string) error {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		expected int
	}{
		{"Root endpoint", "GET", "/", "", 200},
		{"Hello endpoint", "GET", "/hello", "", 200},
		{"Stats endpoint", "GET", "/stats", "", 200},
		{"Config endpoint", "GET", "/config", "", 200},
		{"Echo endpoint", "POST", "/echo", "test data", 200},
		{"Not found", "GET", "/nonexistent", "", 404},
		{"Method not allowed", "PUT", "/hello", "", 405},
	}

	for _, tt := range tests {
		fmt.Printf("  Testing %s... ", tt.name)

		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		request := fmt.Sprintf("%s %s HTTP/1.1\r\n", tt.method, tt.path)
		request += "Host: " + serverAddr + "\r\n"
		if tt.body != "" {
			request += fmt.Sprintf("Content-Length: %d\r\n", len(tt.body))
		}
		request += "\r\n" + tt.body

		_, err = conn.Write([]byte(request))
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to send request: %v", err)
		}

		response := make([]byte, 4096)
		n, err := conn.Read(response)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to read response: %v", err)
		}

		responseStr := string(response[:n])
		if !strings.Contains(responseStr, fmt.Sprintf("HTTP/1.1 %d", tt.expected)) {
			conn.Close()
			return fmt.Errorf("expected status %d, got response: %s", tt.expected, responseStr[:100])
		}

		conn.Close()
		fmt.Println("PASS")
	}

	return nil
}
