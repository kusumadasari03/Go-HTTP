#!/bin/bash
# Simple test script for our HTTP server

echo "🧪 Testing GoHTTP Server"
echo "========================"

# Test different endpoints
echo "📍 Testing root endpoint..."
go run cmd/client/main.go /

echo ""
echo "📍 Testing /hello endpoint..."
go run cmd/client/main.go /hello

echo ""
echo "📍 Testing /info endpoint..."
go run cmd/client/main.go /info

echo ""
echo "📍 Testing /json endpoint..."
go run cmd/client/main.go /json

echo ""
echo "📍 Testing 404 endpoint..."
go run cmd/client/main.go /nonexistent

echo ""
echo "✅ Tests completed!"