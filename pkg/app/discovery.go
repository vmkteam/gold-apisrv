package app

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

type DiscoveryServiceType string

const (
	DiscoveryServiceTypeSync     DiscoveryServiceType = "sync"
	DiscoveryServiceTypeAsync    DiscoveryServiceType = "async"
	DiscoveryServiceTypeExternal DiscoveryServiceType = "external"
)

// DiscoveryManager handles service discovery configuration and endpoints.
// It provides metrics registration and HTTP handlers for discovery information.
type DiscoveryManager struct {
	opts DiscoveryOpts
}

// DiscoveryOpts contains configuration options for service discovery.
type DiscoveryOpts struct {
	DBs            []DiscoveryDB      // Database configurations
	PublicAPI      bool               // Service has public API exposed to internet
	PrivateAPI     bool               // Service has private API exposed to local network
	BrokersrvQueue bool               // Service acts as brokersrv queue
	Cron           bool               // Service use cron
	Services       []DiscoveryService // List of used services
}

type DiscoveryService struct {
	Name string               // service name
	Type DiscoveryServiceType // sync, async, external
}

type DiscoveryDB struct {
	Name        string // database name
	Connections int    // used connections
	Replica     bool   // acts as replica
}

func NewDiscoveryDB(name string, connections int, replica bool) DiscoveryDB {
	return DiscoveryDB{Name: name, Connections: connections, Replica: replica}
}

func NewDiscoveryService(name string, serviceType DiscoveryServiceType) DiscoveryService {
	return DiscoveryService{Name: name, Type: serviceType}
}

func NewDiscoveryManager(opts DiscoveryOpts) *DiscoveryManager {
	return &DiscoveryManager{opts}
}

// Handler returns the discovery configuration as JSON.
func (d *DiscoveryManager) Handler(c echo.Context) error {
	return c.JSON(http.StatusOK, d.opts)
}

func (d *DiscoveryManager) RegisterMetrics() {
	appInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "app",
			Subsystem: "discovery",
			Name:      "service",
			Help:      "App Service Info",
		}, []string{"public_api", "private_api", "cron", "brokersrv_queue"},
	)

	appDBs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "discovery",
			Name:      "db_connections_count_total",
			Help:      "Number of database connections used by App",
		}, []string{"dbname", "replica"},
	)

	appServices := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "app",
			Subsystem: "discovery",
			Name:      "services",
			Help:      "Services used by App",
		}, []string{"service", "type"},
	)

	prometheus.MustRegister(appInfo, appDBs, appServices)

	// add app info
	appInfo.WithLabelValues(strconv.FormatBool(d.opts.PublicAPI),
		strconv.FormatBool(d.opts.PrivateAPI),
		strconv.FormatBool(d.opts.Cron),
		strconv.FormatBool(d.opts.BrokersrvQueue),
	).Set(1)

	// add db conns
	for _, db := range d.opts.DBs {
		appDBs.WithLabelValues(db.Name, strconv.FormatBool(db.Replica)).Add(float64(db.Connections))
	}

	// add direct services
	for _, s := range d.opts.Services {
		appServices.WithLabelValues(s.Name, string(s.Type)).Set(float64(1))
	}
}

// registerDiscovery is a function that registers meta info from service. Must be updated.
func (a *App) registerDiscovery() {
	opts := DiscoveryOpts{
		PublicAPI: true,
		DBs: []DiscoveryDB{
			NewDiscoveryDB(a.cfg.Database.Database, a.cfg.Database.PoolSize, false),
		},
		Services: []DiscoveryService{
			// NewDiscoveryService("", DiscoveryServiceTypeAsync),
		},
	}

	dm := NewDiscoveryManager(opts)
	dm.RegisterMetrics()

	a.echo.GET("/discovery", dm.Handler)
}
