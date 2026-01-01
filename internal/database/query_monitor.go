package database

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"gorm.io/gorm"
)

// QueryPerformancePlugin is a GORM plugin that monitors slow queries
type QueryPerformancePlugin struct {
	SlowQueryThresholdMs int // Threshold in milliseconds for logging slow queries
	Enabled              bool
}

// Name returns the plugin name
func (p *QueryPerformancePlugin) Name() string {
	return "query_performance_monitor"
}

// Initialize initializes the plugin
func (p *QueryPerformancePlugin) Initialize(db *gorm.DB) error {
	// Get slow query threshold from environment variable (default 1000ms)
	thresholdStr := os.Getenv("DB_SLOW_QUERY_THRESHOLD_MS")
	if thresholdStr != "" {
		if threshold, err := strconv.Atoi(thresholdStr); err == nil && threshold > 0 {
			p.SlowQueryThresholdMs = threshold
		}
	}
	if p.SlowQueryThresholdMs == 0 {
		p.SlowQueryThresholdMs = 1000 // Default 1 second
	}

	// Check if query monitoring is enabled (default true in production)
	enabledStr := os.Getenv("DB_QUERY_MONITORING_ENABLED")
	if enabledStr == "" {
		p.Enabled = true // Default enabled
	} else {
		p.Enabled = enabledStr == "true" || enabledStr == "1"
	}

	if !p.Enabled {
		logging.Info("Query performance monitoring is disabled")
		return nil
	}

	// Register callback for after query execution
	err := db.Callback().Query().After("gorm:query").Register("query_performance:after_query", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register query callback: %w", err)
	}

	err = db.Callback().Create().After("gorm:create").Register("query_performance:after_create", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register create callback: %w", err)
	}

	err = db.Callback().Update().After("gorm:update").Register("query_performance:after_update", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register update callback: %w", err)
	}

	err = db.Callback().Delete().After("gorm:delete").Register("query_performance:after_delete", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register delete callback: %w", err)
	}

	logging.WithField("threshold_ms", p.SlowQueryThresholdMs).Info("Query performance monitoring enabled")
	return nil
}

// afterQuery is called after each query execution
func (p *QueryPerformancePlugin) afterQuery(db *gorm.DB) {
	if !p.Enabled {
		return
	}

	// Get query execution duration
	elapsed := time.Since(db.Statement.Context.Value("query_start_time").(time.Time))
	elapsedMs := elapsed.Milliseconds()

	// Log slow queries
	if elapsedMs > int64(p.SlowQueryThresholdMs) {
		sql := db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)

		logging.WithFields(map[string]interface{}{
			"duration_ms": elapsedMs,
			"sql":         sql,
			"rows":        db.Statement.RowsAffected,
			"table":       db.Statement.Table,
		}).Warn("Slow query detected")
	}
}

// InitializeQueryPerformanceMonitoring adds query performance monitoring to the database
func InitializeQueryPerformanceMonitoring(db *gorm.DB) error {
	plugin := &QueryPerformancePlugin{}
	return db.Use(plugin)
}
