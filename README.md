# Owl 🦉

**Production-Grade Observability Toolkit for Go**

`owl` is a unified observability library designed to simplify error handling, logging, metrics, and distributed tracing in Go applications. It provides a cohesive set of types and middleware to ensure your services are production-ready by default.

## 🚀 Features

-   **Unified Error Handling**: `owl.Error` separates internal debug messages (for logs) from public-safe messages (for API responses).
-   **Structured Logging**: Built-in `slog` adapter with support for context propagation and sanitization.
-   **OpenTelemetry Metrics**: Decoupled `OTelAdapter` that works with any OpenTelemetry `MeterProvider` (Prometheus, OTLP, etc.).
-   **Middleware**: robust HTTP and gRPC interceptors for:
    -   Request Logging (latency, status, method)
    -   Panic Recovery
    -   Distributed Trace Injection/Extraction (W3C TraceContext)
    -   Automatic Error Hydration
-   **Production Ready**: Handles numeric status codes, safe defaults, and nil-safe interfaces.

## 📦 Installation

```bash
go get github.com/myuser/owl
```

## 🛠 Usage Guide

### 1. Error Handling

Stop returning simple strings. Use `owl.Problem` to return semantic errors with context.

```go
package main

import (
    "github.com/myuser/owl"
)

func GetUser(id string) error {
    if id == "" {
        // Simple 400 Bad Request
        return owl.Problem(owl.Invalid, owl.WithMsg("user id cannot be empty"))
    }

    // ... database logic fails ...
    err := db.Find(id)
    if err != nil {
        // Return 500 Internal Error
        // - Msg: "db connection failed" (Logged Internally)
        // - SafeMsg: "Something went wrong" (Returned to User)
        // - Err: Wrapped original error
        return owl.Problem(owl.Internal, 
            owl.WithMsg("db connection failed"),
            owl.WithSafeMsg("Something went wrong"),
            owl.WithErr(err),
            owl.WithOp("User.Get"),
        )
    }
    return nil
}
```

### 2. Logging

Use the standard `owl.Logger` interface. The default implementation uses `log/slog`.

```go
import "github.com/myuser/owl/logs"

func main() {
    logger := logs.NewSlogAdapter(nil) // Defaults to JSON handler on stdout
    
    ctx := context.Background()
    logger.Info(ctx, "service started", "port", 8080)
}
```

### 3. Metrics (OpenTelemetry)

`owl` stays out of your way regarding OTel provider configuration. You set up the Exporter (Prometheus, OTLP, stdout), and just pass the `Meter` to owl.

```go
import (
    "github.com/myuser/owl/metrics"
    "go.opentelemetry.io/otel"
)

// ... configure your OTel provider ...
meter := otel.Meter("my-service")
monitor := metrics.NewOTelAdapter(meter)

// Use it
counter := monitor.Counter("requests_total")
counter.Inc(ctx, owl.Attr("type", "api"))
```

### 4. HTTP Middleware

Wrap your handlers to automatically log requests, record metrics, and handle errors.

```go
import "github.com/myuser/owl/middleware"

factory := middleware.NewHTTPFactory(logger, monitor)

http.Handle("/", factory.Wrap(func(w http.ResponseWriter, r *http.Request) error {
    return owl.Problem(owl.NotFound) // Automatically returns 404 JSON response
}))
```

### 5. HTTP Client Middleware

Injects distributed tracing headers and handles error hydration from upstream services.

```go
client := http.Client{
    Transport: middleware.NewHTTPClient(http.DefaultTransport, logger),
}

resp, err := client.Get("http://upstream-service/")
if err != nil {
    // If upstream returned a JSON error, 'err' is already hydrated as *owl.Error
    // containing the remote machine's code and message.
}
```

## 🧩 Architecture

-   **`root`**: Core types (`Error`, `Code`, interfaces).
-   **`logs`**: Logging adapters.
-   **`metrics`**: Metric adapters.
-   **`middleware`**: HTTP/gRPC interceptors.

## 🤝 Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## 📄 License

[MIT](https://choosealicense.com/licenses/mit/)
