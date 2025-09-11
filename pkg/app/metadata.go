package app

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

// registerMetadata is a function that registers meta info from service. Must be updated.
func (a *App) registerMetadata() {
	opts := MetadataOpts{
		HasPublicAPI: true,
		DBs: []DBMetadata{
			NewDBMetadata(a.cfg.Database.Database, a.cfg.Database.PoolSize, false),
		},
		Services: []ServiceMetadata{
			// NewServiceMetadata("srv", MetadataServiceTypeAsync),
		},
	}

	md := NewMetadataManager(opts)
	md.RegisterMetrics()

	a.echo.GET("/debug/metadata", md.Handler)
}

type MetadataServiceType string

const (
	MetadataServiceTypeSync     MetadataServiceType = "sync"
	MetadataServiceTypeAsync    MetadataServiceType = "async"
	MetadataServiceTypeExternal MetadataServiceType = "external"
)

// MetadataManager handles service metadata configuration and endpoints.
// It provides metrics registration and HTTP handlers for metadata information.
type MetadataManager struct {
	opts MetadataOpts
}

// MetadataOpts contains configuration options for service metadata.
type MetadataOpts struct {
	DBs               []DBMetadata      // Database configurations
	HasPublicAPI      bool              // Service has public API exposed to internet
	HasPrivateAPI     bool              // Service has private API exposed to local network
	HasBrokersrvQueue bool              // Service acts as brokersrv queue
	HasCronJobs       bool              // Service use cron
	Services          []ServiceMetadata // List of used services
}

type ServiceMetadata struct {
	Name string              // service name
	Type MetadataServiceType // sync, async, external
}

type DBMetadata struct {
	Name        string // database name
	Connections int    // used connections
	Replica     bool   // acts as replica
}

func NewDBMetadata(name string, connections int, replica bool) DBMetadata {
	return DBMetadata{Name: name, Connections: connections, Replica: replica}
}

func NewServiceMetadata(name string, serviceType MetadataServiceType) ServiceMetadata {
	return ServiceMetadata{Name: name, Type: serviceType}
}

func NewMetadataManager(opts MetadataOpts) *MetadataManager {
	return &MetadataManager{opts}
}

// Handler returns the metadata configuration as JSON.
func (d *MetadataManager) Handler(c echo.Context) error {
	return c.JSON(http.StatusOK, d.opts)
}

func (d *MetadataManager) RegisterMetrics() {
	appInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "app",
			Subsystem: "metadata",
			Name:      "service",
			Help:      "Metadata information about the application service",
		}, []string{"public_api", "private_api", "cron", "brokersrv_queue"},
	)

	appDBs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "metadata",
			Name:      "db_connections_count_total",
			Help:      "Number of database connections used by App",
		}, []string{"dbname", "replica"},
	)

	appServices := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "app",
			Subsystem: "metadata",
			Name:      "services",
			Help:      "Services used by App",
		}, []string{"service", "type"},
	)

	prometheus.MustRegister(appInfo, appDBs, appServices)

	// add app info
	appInfo.WithLabelValues(strconv.FormatBool(d.opts.HasPublicAPI),
		strconv.FormatBool(d.opts.HasPrivateAPI),
		strconv.FormatBool(d.opts.HasCronJobs),
		strconv.FormatBool(d.opts.HasBrokersrvQueue),
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
