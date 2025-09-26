package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"apisrv/pkg/rpc"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vmkteam/appkit"
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
}

// registerDebugHandlers adds /debug/pprof handlers into a.echo instance.
func (a *App) registerDebugHandlers() {
	dbg := a.echo.Group("/debug")

	// add pprof integration
	dbg.Any("/pprof/*", appkit.PprofHandler)

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
		a.echo.GET("/", appkit.RenderRoutes(a.appName, a.echo))
	}
}

// registerAPIHandlers registers main rpc server.
func (a *App) registerAPIHandlers() {
	srv := rpc.New(a.db, a.Logger, a.cfg.Server.IsDevel)
	gen := rpcgen.FromSMD(srv.SMD())

	a.echo.Any("/v1/rpc/", zm.EchoHandler(zm.XRequestID(srv)))
	a.echo.Any("/v1/rpc/doc/", appkit.EchoHandlerFunc(zenrpc.SMDBoxHandler))
	a.echo.Any("/v1/rpc/openrpc.json", appkit.EchoHandlerFunc(rpcgen.Handler(gen.OpenRPC("apisrv", "http://localhost:8075/v1/rpc"))))
	a.echo.Any("/v1/rpc/api.ts", appkit.EchoHandlerFunc(rpcgen.Handler(gen.TSClient(nil))))
}

// registerVTApiHandlers registers vt rpc server.
func (a *App) registerVTApiHandlers() {
	gen := rpcgen.FromSMD(a.vtsrv.SMD())
	tsSettings := typescript.Settings{ExcludedNamespace: []string{NSVFS}, WithClasses: true}

	a.echo.Any("/v1/vt/", zm.EchoHandler(zm.XRequestID(a.vtsrv)))
	a.echo.Any("/v1/vt/doc/", appkit.EchoHandlerFunc(zenrpc.SMDBoxHandler))
	a.echo.Any("/v1/vt/api.ts", appkit.EchoHandlerFunc(rpcgen.Handler(gen.TSCustomClient(tsSettings))))
}
