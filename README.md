# APIWiz Fiber Detect SDK

This SDK provides request/response monitoring and tracing capabilities for Fiber applications integrated with APIWiz.

## Installation

```bash
go get github.com/itorix/apiwiz-go-fiber
```

## Configuration

Add the following configuration to your application:

```go
cfg := &config.Config{
    APIKey:      "your-api-key",
    WorkspaceID: "your-workspace-id",
    DetectAPI:   "your-detect-api-url",
    EnableTracing: true,
    TraceIDHeader: "X-Trace-ID",
    SpanIDHeader:  "X-Span-ID",
    ParentSpanIDHeader: "X-ParentSpan-ID",
    RequestTimestampHeader: "request-timestamp",
    ResponseTimestampHeader: "response-timestamp",
    GatewayTypeHeader: "gateway-type",
}
```

## Usage
x
```go
package main

import (
    "github.com/itorix/apiwiz-go-fiber/pkg/config"
    "github.com/itorix/apiwiz-go-fiber/pkg/middleware"
    "github.com/gofiber/fiber/v2"
)

func main() {
    app := fiber.New()

    cfg := &config.Config{
        // your configuration here
    }

	detect := middleware.NewDetectMiddleware(cfg)
	app.Use(detect.Middleware())

}
```

## Features

- Request/Response monitoring
- Distributed tracing
- Automatic server information collection
- Async compliance data sending
- Configurable headers

## License

MIT






