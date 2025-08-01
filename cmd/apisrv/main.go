package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"apisrv/pkg/app"
	"apisrv/pkg/db"

	"github.com/BurntSushi/toml"
	"github.com/getsentry/sentry-go"
	"github.com/go-pg/pg/v10"
	"github.com/namsral/flag"
	"github.com/vmkteam/embedlog"
)

const appName = "apisrv"

var (
	fs                 = flag.NewFlagSetWithEnvPrefix(os.Args[0], strings.ToUpper(appName), 0)
	flConfigPath       = fs.String("config", "config.toml", "Path to config file")
	flVerbose          = fs.Bool("verbose", false, "enable debug output")
	flJSONLogs         = fs.Bool("json", false, "enable json output")
	flDev              = fs.Bool("dev", false, "enable dev mode")
	flGenerateTSClient = fs.Bool("ts_client", false, "generate TypeScript vt rpc client and exit")
	cfg                app.Config
)

func main() {
	flag.DefaultConfigFlagname = "config.flag"
	exitOnError(fs.Parse(os.Args[1:]))

	// setup logger
	sl, ctx := embedlog.NewLogger(*flVerbose, *flJSONLogs), context.Background()
	if *flDev {
		sl = embedlog.NewDevLogger()
	}
	slog.SetDefault(sl.Log()) // set default logger
	ql := db.NewQueryLogger(sl)
	pg.SetLogger(ql)

	version := appVersion()
	sl.Print(ctx, "starting", "app", appName, "version", version)
	if _, err := toml.DecodeFile(*flConfigPath, &cfg); err != nil {
		exitOnError(err)
	}

	// enable sentry
	if cfg.Sentry.DSN != "" {
		exitOnError(sentry.Init(sentry.ClientOptions{
			Dsn:         cfg.Sentry.DSN,
			Environment: cfg.Sentry.Environment,
			Release:     version,
		}))
	}

	// check db connection
	pgdb := pg.Connect(cfg.Database)
	dbc := db.New(pgdb)

	v, err := dbc.Version()
	exitOnError(err)
	sl.Print(ctx, "connected to db", "version", v)

	// log all sql queries
	if *flDev {
		pgdb.AddQueryHook(ql)
	}

	// create & run app
	a := app.New(appName, sl, cfg, dbc, pgdb)

	// enable vfs
	if cfg.Server.EnableVFS {
		err = a.RegisterVFS(cfg.VFS)
		exitOnError(err)
	}

	// generate TS client from cmd flags
	if *flGenerateTSClient {
		b, err := a.VTTypeScriptClient()
		exitOnError(err)
		_, _ = fmt.Fprint(os.Stdout, string(b))
		os.Exit(0)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// run app and send panic to sentry
	go func() {
		defer func() {
			if err := recover(); err != nil {
				sentry.CurrentHub().Recover(err)
				sentry.Flush(time.Second * 3)
				panic(err)
			}
		}()

		if err := a.Run(ctx); err != nil {
			a.Print(ctx, "shutting down http server", "err", err)
		}
	}()
	<-quit
	a.Shutdown(5 * time.Second)
}

// exitOnError calls log.Fatal if err wasn't nil.
func exitOnError(err error) {
	if err != nil {
		//nolint:sloglint,noctx
		slog.Error(err.Error())
		os.Exit(1)
	}
}

// appVersion returns app version from VCS info.
func appVersion() string {
	result := "devel"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			result = v.Value
		}
	}

	if len(result) > 8 {
		result = result[:8]
	}

	return result
}
