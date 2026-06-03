package http

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// Request represents an HTTP request
type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    string
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// ParseRequestWithConfig parses raw HTTP request data with configuration limits
func ParseRequestWithConfig(conn net.Conn, maxBodySize int) (*Request, error) {
	reader := bufio.NewReader(conn)

	// Read the request line (GET /path HTTP/1.1)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %v", err)
	}
	requestLine = strings.TrimSpace(requestLine)

	// Parse request line
	parts := strings.Fields(requestLine)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line format")
	}

	req := &Request{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	// Parse headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %v", err)
		}
		line = strings.TrimSpace(line)

		// Empty line indicates end of headers
		if len(line) == 0 {
			break
		}

		// Parse header (Key: Value)
		headerStr := line
		colonIndex := strings.Index(headerStr, ":")
		if colonIndex == -1 {
			continue // Skip malformed headers
		}

		key := strings.TrimSpace(headerStr[:colonIndex])
		value := strings.TrimSpace(headerStr[colonIndex+1:])
		req.Headers[strings.ToLower(key)] = value
	}

	// Parse body if Content-Length is present
	if contentLength, exists := req.Headers["content-length"]; exists {
		// Parse content length
		var length int
		fmt.Sscanf(contentLength, "%d", &length)

		if length > maxBodySize {
			return nil, fmt.Errorf("request body too large: %d bytes (max: %d)", length, maxBodySize)
		}

		if length > 0 {
			bodyBytes := make([]byte, length)
			_, err := io.ReadFull(reader, bodyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %v", err)
			}
			req.Body = string(bodyBytes)
		}
	}

	return req, nil
}

// ParseRequest parses raw HTTP request data into a Request struct (with default limits)
func ParseRequest(conn net.Conn) (*Request, error) {
	return ParseRequestWithConfig(conn, 1024*1024) // 1MB default limit
}

// WriteResponse writes an HTTP response to the connection
func (r *Response) WriteResponse(conn net.Conn) error {
	// Status line
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.StatusCode, r.StatusText)

	// Headers
	headers := ""
	for key, value := range r.Headers {
		headers += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	// Add Content-Length if not present
	if _, exists := r.Headers["Content-Length"]; !exists {
		headers += fmt.Sprintf("Content-Length: %d\r\n", len(r.Body))
	}

	// Empty line separates headers from body
	response := statusLine + headers + "\r\n" + r.Body

	_, err := conn.Write([]byte(response))
	return err
}

// NewResponse creates a new HTTP response
func NewResponse(statusCode int, statusText string) *Response {
	return &Response{
		StatusCode: statusCode,
		StatusText: statusText,
		Headers:    make(map[string]string),
		Body:       "",
	}
}

// Common response creators for convenience
func NewOKResponse(body string) *Response {
	resp := NewResponse(200, "OK")
	resp.Body = body
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func NewNotFoundResponse() *Response {
	resp := NewResponse(404, "Not Found")
	resp.Body = "404 - Page Not Found"
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func NewBadRequestResponse(message string) *Response {
	resp := NewResponse(400, "Bad Request")
	if message == "" {
		message = "400 - Bad Request"
	}
	resp.Body = message
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func NewMethodNotAllowedResponse() *Response {
	resp := NewResponse(405, "Method Not Allowed")
	resp.Body = "405 - Method Not Allowed"
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func NewInternalErrorResponse() *Response {
	resp := NewResponse(500, "Internal Server Error")
	resp.Body = "500 - Internal Server Error"
	resp.Headers["Content-Type"] = "text/plain"
	return resp
}

func NewJSONResponse(body string) *Response {
	resp := NewResponse(200, "OK")
	resp.Body = body
	resp.Headers["Content-Type"] = "application/json"
	return resp
}

// GetMimeType returns the MIME type based on file extension
func GetMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	case ".xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

// ServeStaticFile serves a static file from the filesystem
func ServeStaticFile(filePath string) *Response {
	// Check if file exists
	file, err := os.Open(filePath)
	if err != nil {
		return NewNotFoundResponse()
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return NewInternalErrorResponse()
	}

	// Create response
	resp := NewResponse(200, "OK")
	resp.Body = string(content)
	resp.Headers["Content-Type"] = GetMimeType(filePath)

	return resp
}
