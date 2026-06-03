package tests

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/aakifnehal/gohttp/internal/http"
)

// Helper function to create a test connection with request data
func createTestRequest(requestData string) (net.Conn, net.Conn, error) {
	// Create a pipe to simulate network connection
	server, client := net.Pipe()

	// Send request data to the pipe
	go func() {
		defer client.Close()
		client.Write([]byte(requestData))
	}()

	return server, client, nil
}

func TestHTTPRequestParsing(t *testing.T) {
	tests := []struct {
		name           string
		requestData    string
		expectedMethod string
		expectedPath   string
		expectedBody   string
		shouldFail     bool
	}{
		{
			name: "Simple GET request",
			requestData: "GET /hello HTTP/1.1\r\n" +
				"Host: localhost:8080\r\n" +
				"User-Agent: test\r\n" +
				"\r\n",
			expectedMethod: "GET",
			expectedPath:   "/hello",
			expectedBody:   "",
			shouldFail:     false,
		},
		{
			name: "POST request with body",
			requestData: "POST /echo HTTP/1.1\r\n" +
				"Host: localhost:8080\r\n" +
				"Content-Length: 12\r\n" +
				"\r\n" +
				"Hello World!",
			expectedMethod: "POST",
			expectedPath:   "/echo",
			expectedBody:   "Hello World!",
			shouldFail:     false,
		},
		{
			name: "Invalid request line",
			requestData: "INVALID REQUEST\r\n" +
				"\r\n",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, err := createTestRequest(tt.requestData)
			if err != nil {
				t.Fatalf("Failed to create test request: %v", err)
			}
			defer server.Close()

			req, err := http.ParseRequest(server)
			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected parsing to fail, but it succeeded")
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to parse request: %v", err)
			}

			if req.Method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, req.Method)
			}

			if req.Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, req.Path)
			}

			if req.Body != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, req.Body)
			}
		})
	}
}

func TestHTTPResponseGeneration(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		statusText     string
		body           string
		headers        map[string]string
		expectedOutput string
	}{
		{
			name:       "Simple OK response",
			statusCode: 200,
			statusText: "OK",
			body:       "Hello World",
			headers:    map[string]string{"Content-Type": "text/plain"},
			expectedOutput: "HTTP/1.1 200 OK\r\n" +
				"Content-Type: text/plain\r\n" +
				"Content-Length: 11\r\n" +
				"\r\n" +
				"Hello World",
		},
		{
			name:       "JSON response",
			statusCode: 200,
			statusText: "OK",
			body:       `{"message":"test"}`,
			headers:    map[string]string{"Content-Type": "application/json"},
			expectedOutput: "HTTP/1.1 200 OK\r\n" +
				"Content-Type: application/json\r\n" +
				"Content-Length: 18\r\n" +
				"\r\n" +
				`{"message":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				StatusText: tt.statusText,
				Headers:    tt.headers,
				Body:       tt.body,
			}

			var buf bytes.Buffer
			// Create a fake connection that writes to our buffer
			conn := &fakeConn{writer: &buf}

			err := resp.WriteResponse(conn)
			if err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}

			output := buf.String()
			if output != tt.expectedOutput {
				t.Errorf("Expected output:\n%q\nGot:\n%q", tt.expectedOutput, output)
			}
		})
	}
}

func TestMimeTypeDetection(t *testing.T) {
	tests := []struct {
		filename     string
		expectedMime string
	}{
		{"test.html", "text/html"},
		{"test.css", "text/css"},
		{"test.js", "application/javascript"},
		{"test.json", "application/json"},
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.txt", "text/plain"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			mime := http.GetMimeType(tt.filename)
			if mime != tt.expectedMime {
				t.Errorf("Expected MIME type %s for %s, got %s", tt.expectedMime, tt.filename, mime)
			}
		})
	}
}

func TestResponseCreators(t *testing.T) {
	// Test OK response
	okResp := http.NewOKResponse("test body")
	if okResp.StatusCode != 200 || okResp.StatusText != "OK" {
		t.Errorf("NewOKResponse failed: got %d %s", okResp.StatusCode, okResp.StatusText)
	}

	// Test Not Found response
	notFoundResp := http.NewNotFoundResponse()
	if notFoundResp.StatusCode != 404 || notFoundResp.StatusText != "Not Found" {
		t.Errorf("NewNotFoundResponse failed: got %d %s", notFoundResp.StatusCode, notFoundResp.StatusText)
	}

	// Test JSON response
	jsonResp := http.NewJSONResponse(`{"test": true}`)
	if jsonResp.Headers["Content-Type"] != "application/json" {
		t.Errorf("NewJSONResponse failed: Content-Type is %s", jsonResp.Headers["Content-Type"])
	}
}

// Benchmark HTTP request parsing
func BenchmarkRequestParsing(b *testing.B) {
	requestData := "GET /test HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"User-Agent: benchmark\r\n" +
		"\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server, _, _ := createTestRequest(requestData)
		http.ParseRequest(server)
		server.Close()
	}
}

// Benchmark response generation
func BenchmarkResponseGeneration(b *testing.B) {
	resp := http.NewOKResponse("Benchmark test response body")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		conn := &fakeConn{writer: &buf}
		resp.WriteResponse(conn)
	}
}

// fakeConn implements net.Conn for testing
type fakeConn struct {
	writer io.Writer
	reader io.Reader
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	if f.reader != nil {
		return f.reader.Read(b)
	}
	return 0, io.EOF
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	if f.writer != nil {
		return f.writer.Write(b)
	}
	return len(b), nil
}

func (f *fakeConn) Close() error                       { return nil }
func (f *fakeConn) LocalAddr() net.Addr                { return nil }
func (f *fakeConn) RemoteAddr() net.Addr               { return nil }
func (f *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

// Integration test function that can be run manually
func RunIntegrationTests(serverAddr string) error {
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
		{"Echo endpoint", "POST", "/echo", "test data", 200},
		{"Not found", "GET", "/nonexistent", "", 404},
		{"Method not allowed", "PUT", "/hello", "", 405},
	}

	for _, tt := range tests {
		fmt.Printf("Running integration test: %s... ", tt.name)

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
		if err != nil && err != io.EOF {
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
