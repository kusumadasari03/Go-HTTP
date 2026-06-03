# HTTP Protocol Fundamentals

This document explains the HTTP/1.1 protocol basics as implemented in this project.

## HTTP Request Format

An HTTP request consists of:

1. **Request Line**: `METHOD PATH VERSION`
   ```
   GET /hello HTTP/1.1
   ```

2. **Headers**: Key-value pairs separated by colons
   ```
   Host: localhost:8080
   User-Agent: GoHTTP-TestClient/1.0
   Content-Type: application/json
   ```

3. **Empty Line**: Separates headers from body

4. **Body**: Optional request data (for POST, PUT, etc.)

## HTTP Response Format

An HTTP response consists of:

1. **Status Line**: `VERSION STATUS_CODE STATUS_TEXT`
   ```
   HTTP/1.1 200 OK
   ```

2. **Headers**: Response metadata
   ```
   Content-Type: text/plain
   Content-Length: 13
   ```

3. **Empty Line**: Separates headers from body

4. **Body**: Response data

## Common Status Codes

- **200 OK**: Request successful
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Server error

## HTTP Methods

- **GET**: Retrieve data
- **POST**: Submit data
- **PUT**: Update data
- **DELETE**: Remove data

## Example HTTP Exchange

**Request:**
```
GET /hello HTTP/1.1
Host: localhost:8080
User-Agent: GoHTTP-TestClient/1.0

```

**Response:**
```
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 13

Hello, World!
```

## Implementation Notes

Our server implements HTTP/1.1 parsing by:

1. Reading the request line
2. Parsing headers until empty line
3. Creating structured Request object
4. Generating appropriate Response object
5. Writing formatted response to connection