package middleware

import (
	"context"
	"encoding/json"

	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
)

const (
	isDevelCtx      contextKey = "isDevel"
	ctxPlatformKey  contextKey = "platform"
	ctxVersionKey   contextKey = "version"
	ctxMethodKey    contextKey = "method"
	ctxIPKey        contextKey = "ip"
	ctxUserAgentKey contextKey = "userAgent"
	ctxCountryKey   contextKey = "country"

	ctxNotificationKey = "JSONRPC2-Notification"

	// DefaultServerName is a global default name, mostly used for metrics.
	DefaultServerName = ""

	maxUserAgentLength = 2048
	maxVersionLength   = 64
	maxCountryLength   = 16
)

type (
	contextKey string
	Printf     func(format string, v ...interface{})
	Print      func(ctx context.Context, msg string, args ...any)
	LogAttrs   func(ctx context.Context, r zenrpc.Response) []any
)

// WithDevel sets bool flag to context for detecting development environment.
func WithDevel(isDevel bool) zenrpc.MiddlewareFunc {
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			ctx = NewIsDevelContext(ctx, isDevel)
			return h(ctx, method, params)
		}
	}
}

// WithNoCancelContext ignores Cancel func from context. This is useful for passing context to `go-pg`.
func WithNoCancelContext() zenrpc.MiddlewareFunc {
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			return h(context.WithoutCancel(ctx), method, params)
		}
	}
}

// WithHeaders sets User-Agent, Platform, Version, X-Country headers to context. User-Agent strips to 2048 chars, Platform and Version – to 64, X-Country - to 16.
func WithHeaders() zenrpc.MiddlewareFunc {
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			if req, ok := zenrpc.RequestFromContext(ctx); ok && req != nil {
				ctx = NewUserAgentContext(ctx, req.UserAgent())
				ctx = NewPlatformContext(ctx, req.Header.Get("Platform"))
				ctx = NewVersionContext(ctx, req.Header.Get("Version"))
				ctx = NewXRequestIDContext(ctx, req.Header.Get(echo.HeaderXRequestID))
				ctx = NewCountryContext(ctx, req.Header.Get("X-Country"))
				ctx = NewMethodContext(ctx, method)
			}
			return h(ctx, method, params)
		}
	}
}

// NewIPContext creates new context with IP.
func NewIPContext(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ctxIPKey, ip)
}

// IPFromContext returns IP from context.
func IPFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxIPKey).(string)
	return r
}

// NewUserAgentContext creates new context with User-Agent.
func NewUserAgentContext(ctx context.Context, ua string) context.Context {
	return context.WithValue(ctx, ctxUserAgentKey, cutString(ua, maxUserAgentLength))
}

// UserAgentFromContext returns userAgent from context.
func UserAgentFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxUserAgentKey).(string)
	return r
}

// NewNotificationContext creates new context with JSONRPC2 notification flag.
func NewNotificationContext(ctx context.Context) context.Context {
	//nolint:staticcheck // must be global
	return context.WithValue(ctx, ctxNotificationKey, true)
}

// NotificationFromContext returns JSONRPC2 notification flag from context.
func NotificationFromContext(ctx context.Context) bool {
	r, _ := ctx.Value(ctxNotificationKey).(bool)
	return r
}

// NewIsDevelContext creates new context with isDevel flag.
func NewIsDevelContext(ctx context.Context, isDevel bool) context.Context {
	return context.WithValue(ctx, isDevelCtx, isDevel)
}

// IsDevelFromContext returns isDevel flag from context.
func IsDevelFromContext(ctx context.Context) bool {
	if isDevel, ok := ctx.Value(isDevelCtx).(bool); ok {
		return isDevel
	}
	return false
}

// NewPlatformContext creates new context with platform.
func NewPlatformContext(ctx context.Context, platform string) context.Context {
	return context.WithValue(ctx, ctxPlatformKey, cutString(platform, 64))
}

// PlatformFromContext returns platform from context.
func PlatformFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxPlatformKey).(string)
	return r
}

// NewVersionContext creates new context with version.
func NewVersionContext(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, ctxVersionKey, cutString(version, maxVersionLength))
}

// VersionFromContext returns version from context.
func VersionFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxVersionKey).(string)
	return r
}

// NewCountryContext creates new context with country.
func NewCountryContext(ctx context.Context, country string) context.Context {
	return context.WithValue(ctx, ctxCountryKey, cutString(country, maxCountryLength))
}

// CountryFromContext returns country from context.
func CountryFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxCountryKey).(string)
	return r
}

// NewMethodContext creates new context with Method.
func NewMethodContext(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ctxMethodKey, method)
}

// MethodFromContext returns Method from context.
func MethodFromContext(ctx context.Context) string {
	r, _ := ctx.Value(ctxMethodKey).(string)
	return r
}

// cutString cuts string with given length.
func cutString(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}

// fullMethodName returns namespace.method or serverName.namespace.method.
func fullMethodName(serverName, namespace, method string) string {
	name := namespace + "." + method
	if serverName != "" {
		name = serverName + "." + name
	}

	return name
}
