package middleware

import (
	"net/http"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
)

// CORS allows certain CORS headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Authorization2, Origin, X-Requested-With, Content-Type, Accept, Platform, Version, X-Request-ID")
		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// XRequestID add X-Request-ID header if not exists.
func XRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(echo.HeaderXRequestID)
		if !isValidXRequestID(requestID) {
			requestID = generateXRequestID()
			r.Header.Add(echo.HeaderXRequestID, requestID)
		}
		w.Header().Set(echo.HeaderXRequestID, requestID)

		next.ServeHTTP(w, r)
	})
}

// EchoHandler is wrapper for Echo.
func EchoHandler(next http.Handler) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx = applySentryHubToContext(ctx)
		ctx = applyIPToContext(ctx)
		req := ctx.Request()
		CORS(XRequestID(next)).ServeHTTP(ctx.Response(), req)
		return nil
	}
}

// EchoSentryHubContext middleware applies sentry hub to context for zenrpc middleware.
func EchoSentryHubContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(applySentryHubToContext(c))
		}
	}
}

// EchoIPContext middleware applies client ip to context for zenrpc middleware.
func EchoIPContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(applyIPToContext(c))
		}
	}
}

func applySentryHubToContext(c echo.Context) echo.Context {
	if hub := sentryecho.GetHubFromContext(c); hub != nil {
		req := c.Request()
		c.SetRequest(req.WithContext(NewSentryHubContext(req.Context(), hub)))
	}
	return c
}

func applyIPToContext(c echo.Context) echo.Context {
	req := c.Request()
	c.SetRequest(req.WithContext(NewIPContext(req.Context(), c.RealIP())))
	return c
}
