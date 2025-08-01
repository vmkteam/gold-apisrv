package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"strings"

	"apisrv/pkg/rpc"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vmkteam/rpcgen/v2"
	"github.com/vmkteam/rpcgen/v2/typescript"
	zm "github.com/vmkteam/zenrpc-middleware"
	"github.com/vmkteam/zenrpc/v2"
)

// runHTTPServer is a function that starts http listener using labstack/echo.
func (a *App) runHTTPServer(ctx context.Context, host string, port int) error {
	listenAddress := fmt.Sprintf("%s:%d", host, port)
	addr := "http://" + listenAddress
	a.Print(ctx, "starting http listener", "url", addr, "smdbox", addr+"/v1/rpc/doc/")

	return a.echo.Start(listenAddress)
}

// registerHandlers register echo handlers.
func (a *App) registerHandlers() {
	a.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders: []string{"Authorization", "Authorization2", "Origin", "X-Requested-With", "Content-Type", "Accept", "Platform", "Version"},
	}))

	// sentry middleware
	a.echo.Use(sentryecho.New(sentryecho.Options{
		Repanic:         true,
		WaitForDelivery: true,
	}))

	a.echo.Use(zm.EchoIPContext(), zm.EchoSentryHubContext())
}

// registerDebugHandlers adds /debug/pprof handlers into a.echo instance.
func (a *App) registerDebugHandlers() {
	dbg := a.echo.Group("/debug")

	// add pprof integration
	dbg.Any("/pprof/*", func(c echo.Context) error {
		if h, p := http.DefaultServeMux.Handler(c.Request()); p != "" {
			h.ServeHTTP(c.Response(), c.Request())
			return nil
		}
		return echo.NewHTTPError(http.StatusNotFound)
	})

	// add healthcheck
	a.echo.GET("/status", func(c echo.Context) error {
		// test postgresql connection
		err := a.db.Ping(c.Request().Context())
		if err != nil {
			a.Error(c.Request().Context(), "failed to check db connection", "err", err)
			return c.String(http.StatusInternalServerError, "DB error")
		}
		return c.String(http.StatusOK, "OK")
	})

	// show all routes in devel mode
	if a.cfg.Server.IsDevel {
		a.echo.GET("/", a.renderRouters)
	}
}

// registerAPIHandlers registers main rpc server.
func (a *App) registerAPIHandlers() {
	srv := rpc.New(a.db, a.Logger, a.cfg.Server.IsDevel)
	gen := rpcgen.FromSMD(srv.SMD())

	a.echo.Any("/v1/rpc/", zm.EchoHandler(zm.XRequestID(srv)))
	a.echo.Any("/v1/rpc/doc/", echo.WrapHandler(http.HandlerFunc(zenrpc.SMDBoxHandler)))
	a.echo.Any("/v1/rpc/openrpc.json", echo.WrapHandler(http.HandlerFunc(rpcgen.Handler(gen.OpenRPC("apisrv", "http://localhost:8075/v1/rpc")))))
	a.echo.Any("/v1/rpc/api.ts", echo.WrapHandler(http.HandlerFunc(rpcgen.Handler(gen.TSClient(nil)))))
}

// registerVTApiHandlers registers vt rpc server.
func (a *App) registerVTApiHandlers() {
	gen := rpcgen.FromSMD(a.vtsrv.SMD())
	tsSettings := typescript.Settings{ExcludedNamespace: []string{NSVFS}, WithClasses: true}

	a.echo.Any("/v1/vt/", zm.EchoHandler(zm.XRequestID(a.vtsrv)))
	a.echo.Any("/v1/vt/doc/", echo.WrapHandler(http.HandlerFunc(zenrpc.SMDBoxHandler)))
	a.echo.Any("/v1/vt/api.ts", echo.WrapHandler(http.HandlerFunc(rpcgen.Handler(gen.TSCustomClient(tsSettings)))))
}

// renderRoutes is a simple echo handler that renders all routes as HTML.
func (a *App) renderRouters(ctx echo.Context) error {
	// collect paths
	routesByPaths := make(map[string]struct{})
	var paths []string
	for _, route := range a.echo.Routes() {
		if _, ok := routesByPaths[route.Path]; !ok {
			routesByPaths[route.Path] = struct{}{}
			paths = append(paths, strings.TrimRight(route.Path, "*"))
		}
	}
	sort.Strings(paths)

	// render template
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<html><body><h1>%s</h1><ul>", a.appName))
	for _, path := range paths {
		sb.WriteString(fmt.Sprintf(`<li><a href="%s">%s</a></li>`, path, path))
	}
	sb.WriteString("</ul></body></html>")

	return ctx.HTML(http.StatusOK, sb.String())
}
