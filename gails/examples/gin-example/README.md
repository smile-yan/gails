# Gin Example

This example demonstrates how to use the [Gin web framework](https://github.com/gin-gonic/gin) with Gails.

## Overview

This example shows how to:

- Set up Gin as the asset handler for a Gails application
- Create a middleware that routes requests between Gails and Gin
- Define API endpoints with Gin
- Communicate between the Gin-served frontend and Gails backend
- Implement custom Gin middleware

## Running the Example

```bash
cd v3/examples/gin-example
go mod tidy
go run .
```

## How It Works

The example uses Gin's HTTP router to serve the frontend content whilst still allowing Gails to handle its internal routes. This is achieved through:

1. Creating a Gin router with routes for the frontend
2. Implementing a middleware function that decides whether to pass requests to Gin or let Gails handle them
3. Configuring the Gails application to use both the Gin router as the asset handler and the custom middleware

### Gails-Gin Integration

The key part of the integration is the middleware function:

```go
func GinMiddleware(ginEngine *gin.Engine) application.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Let Gails handle its internal routes
            if r.URL.Path == "/gails/runtime.js" || r.URL.Path == "/gails/ipc" {
                next.ServeHTTP(w, r)
                return
            }

            // Let Gin handle everything else
            ginEngine.ServeHTTP(w, r)
        })
    }
}
```

This allows you to leverage Gin's powerful routing and middleware capabilities whilst still maintaining full access to Gails features.

### Custom Gin Middleware

The example also demonstrates how to create custom Gin middleware:

```go
func LoggingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Start timer
        startTime := time.Now()

        // Process request
        c.Next()

        // Calculate latency
        latency := time.Since(startTime)

        // Log request details
        log.Printf("[GIN] %s | %s | %s | %d | %s",
            c.Request.Method,
            c.Request.URL.Path,
            c.ClientIP(),
            c.Writer.Status(),
            latency,
        )
    }
}
```

This middleware is applied to all Gin routes and logs details about each request.

### Application Configuration

The Gails application is configured to use Gin as follows:

```go
app := application.New(application.Options{
    Name:        "Gin Example",
    Description: "A demo of using Gin with Gails",
    Mac: application.MacOptions{
        ApplicationShouldTerminateAfterLastWindowClosed: true,
    },
    Assets: application.AssetOptions{
        Handler:    ginEngine,
        Middleware: GinMiddleware(ginEngine),
    },
})
```

This configuration tells Gails to:
1. Use the Gin engine as the primary handler for HTTP requests
2. Use our custom middleware to route requests between Gails and Gin
