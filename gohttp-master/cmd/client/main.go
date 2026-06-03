package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/client/main.go <path>")
		fmt.Println("Example: go run cmd/client/main.go /hello")
		os.Exit(1)
	}

	path := os.Args[1]

	// Connect to our server
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Printf("❌ Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Send HTTP request
	request := fmt.Sprintf("GET %s HTTP/1.1\r\n", path) +
		"Host: 127.0.0.1:8080\r\n" +
		"User-Agent: GoHTTP-TestClient/1.0\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	fmt.Printf("📤 Sending request to %s\n", path)
	_, err = conn.Write([]byte(request))
	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		os.Exit(1)
	}

	// Read response
	response, err := io.ReadAll(conn)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📥 Response:\n%s\n", string(response))
}
