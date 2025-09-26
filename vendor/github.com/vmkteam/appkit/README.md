# AppKit – Go Application Framework Utilities

[![Linter Status](https://github.com/vmkteam/appkit/actions/workflows/golangci-lint.yml/badge.svg?branch=master)](https://github.com/vmkteam/appkit/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/vmkteam/appkit)](https://goreportcard.com/report/github.com/vmkteam/appkit)
[![Go Reference](https://pkg.go.dev/badge/github.com/vmkteam/appkit.svg)](https://pkg.go.dev/github.com/vmkteam/appkit)

A comprehensive Go utility package providing common application framework components for building production-ready HTTP services.

## Features

### Core Utilities
- **Version Management**: Automatically extracts version information from VCS (Git) build info
- **HTTP Headers**: Pre-configured internal service headers with app name and version

### Echo Framework Integration
- **Pre-configured Echo Server**: Ready-to-use Echo instance with security and monitoring middleware
- **IP Extraction**: Proper IP handling with real IP header support and CIDR trust ranges
- **Sentry Integration**: Built-in error tracking with Sentry middleware
- **Route Rendering**: Automatic HTML route listing for service discovery

### Monitoring & Metrics
- **Prometheus Integration**: Comprehensive HTTP metrics (request count, response duration)
- **Metadata Management**: Service configuration tracking and metrics registration
- **pprof Support**: Built-in profiling endpoint handler

### Service Metadata
- **Service Discovery**: Track internal and external service dependencies
- **Database Monitoring**: Connection tracking and replica status
- **API Exposure**: Public/private API configuration tracking

## Usage Examples

Please see example/main.go

### Basic HTTP Server with Metrics

```go
e := appkit.NewEcho()

// Add Prometheus metrics
e.Use(appkit.HTTPMetrics("api-service"))

// Route rendering for service discovery
e.GET("/", appkit.RenderRoutes("API Service", e))

// pprof endpoints
e.GET("/debug/pprof/*", appkit.PprofHandler)
```

### Service Metadata Configuration

```go
metadata := appkit.NewMetadataManager(appkit.MetadataOpts{
    DBs: []appkit.DBMetadata{
        appkit.NewDBMetadata("postgres", 10, false),
        appkit.NewDBMetadata("redis", 5, true),
    },
    HasPublicAPI:      true,
    HasPrivateAPI:     false,
    HasBrokersrvQueue: true,
    HasCronJobs:       true,
    Services: []appkit.ServiceMetadata{
        appkit.NewServiceMetadata("auth-service", appkit.MetadataServiceTypeSync),
        appkit.NewServiceMetadata("payment-service", appkit.MetadataServiceTypeExternal),
    },
})

// Register metrics
metadata.RegisterMetrics()

// Add metadata endpoint
e.GET("/metadata", metadata.Handler)
```

### Internal Service Communication

```go
func callInternalService() {
    headers := appkit.NewInternalHeaders("worker-service", appkit.Version())
    
    req, _ := http.NewRequest("GET", "http://api-service/internal/data", nil)
    req.Header = headers
    
    // Make request with proper internal headers
    client.Do(req)
}
```

## API Reference

### Core Functions

- `Version() string` - Returns VCS revision or "devel"
- `NewInternalHeaders(appName, version string) http.Header` - Creates standardized internal headers

### Echo Framework

- `NewEcho() *echo.Echo` - Creates pre-configured Echo instance
- `RenderRoutes(appName string, e *echo.Echo) echo.HandlerFunc` - HTML route renderer
- `PprofHandler(c echo.Context) error` - pprof endpoint handler
- `EchoHandlerFunc(next http.HandlerFunc) echo.HandlerFunc` - HTTP handler wrapper

### Metrics & Monitoring

- `HTTPMetrics(serverName string) echo.MiddlewareFunc` - Prometheus HTTP metrics middleware
- `MetadataManager` - Service metadata configuration and metrics

## Dependencies

- [Echo](https://echo.labstack.com/) - High performance HTTP framework
- [Prometheus](https://prometheus.io/) - Metrics collection and monitoring
- [Sentry](https://sentry.io/) - Error tracking and monitoring
- [zenrpc-middleware](https://github.com/vmkteam/zenrpc-middleware) - RPC middleware utilities

## Metrics Exported

### HTTP Metrics
- `app_http_requests_total` - Total HTTP requests by method/path/status
- `app_http_responses_duration_seconds` - Response time distribution

### Service Metadata Metrics
- `app_metadata_service` - Service configuration information
- `app_metadata_db_connections_total` - Database connection counts
- `app_metadata_services` - Service dependencies

## License

MIT License - see LICENSE file for details.