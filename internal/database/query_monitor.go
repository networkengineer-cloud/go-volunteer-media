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

	// Register "before" callbacks to capture start time for each operation type
	err := db.Callback().Query().Before("gorm:query").Register("query_performance:before_query", p.beforeQuery)
	if err != nil {
		return fmt.Errorf("failed to register before query callback: %w", err)
	}

	err = db.Callback().Create().Before("gorm:create").Register("query_performance:before_create", p.beforeQuery)
	if err != nil {
		return fmt.Errorf("failed to register before create callback: %w", err)
	}

	err = db.Callback().Update().Before("gorm:update").Register("query_performance:before_update", p.beforeQuery)
	if err != nil {
		return fmt.Errorf("failed to register before update callback: %w", err)
	}

	err = db.Callback().Delete().Before("gorm:delete").Register("query_performance:before_delete", p.beforeQuery)
	if err != nil {
		return fmt.Errorf("failed to register before delete callback: %w", err)
	}

	// Register "after" callbacks to measure duration and log slow queries
	err = db.Callback().Query().After("gorm:query").Register("query_performance:after_query", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register after query callback: %w", err)
	}

	err = db.Callback().Create().After("gorm:create").Register("query_performance:after_create", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register after create callback: %w", err)
	}

	err = db.Callback().Update().After("gorm:update").Register("query_performance:after_update", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register after update callback: %w", err)
	}

	err = db.Callback().Delete().After("gorm:delete").Register("query_performance:after_delete", p.afterQuery)
	if err != nil {
		return fmt.Errorf("failed to register after delete callback: %w", err)
	}

	logging.WithField("threshold_ms", p.SlowQueryThresholdMs).Info("Query performance monitoring enabled")
	return nil
}

// beforeQuery is called before each query execution to capture start time
func (p *QueryPerformancePlugin) beforeQuery(db *gorm.DB) {
	if !p.Enabled {
		return
	}
	// Store start time in the statement context
	db.InstanceSet("query_start_time", time.Now())
}

// afterQuery is called after each query execution
func (p *QueryPerformancePlugin) afterQuery(db *gorm.DB) {
	if !p.Enabled {
		return
	}

	// Get query execution duration
	startTime, exists := db.InstanceGet("query_start_time")
	if !exists {
		// If start time wasn't set, skip logging (shouldn't happen with proper before callback)
		return
	}

	start, ok := startTime.(time.Time)
	if !ok {
		return
	}

	elapsed := time.Since(start)
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
