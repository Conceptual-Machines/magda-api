package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	startTime time.Time
	version   string
}

func NewMetricsHandler(version string) *MetricsHandler {
	return &MetricsHandler{
		startTime: time.Now(),
		version:   version,
	}
}

const (
	secondsPerMinute = 60
	secondsPerHour   = 3600
)

// formatUptime formats the uptime duration with seconds rounded to 2 decimal places
func formatUptime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % secondsPerMinute
	seconds := d.Seconds() - float64(hours*secondsPerHour) - float64(minutes*secondsPerMinute)

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%.2fs", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%.2fs", minutes, seconds)
	}
	return fmt.Sprintf("%.2fs", seconds)
}

type MetricsResponse struct {
	Status    string                 `json:"status"`
	Uptime    string                 `json:"uptime"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	StartTime string                 `json:"start_time"`
	System    SystemMetrics          `json:"system"`
	API       map[string]interface{} `json:"api"`
}

type SystemMetrics struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAllocMB   uint64 `json:"mem_alloc_mb"`
	MemTotalMB   uint64 `json:"mem_total_mb"`
	NumGC        uint32 `json:"num_gc"`
}

const (
	bytesToMB = 1024 * 1024
)

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime)

	metrics := MetricsResponse{
		Status:    "healthy",
		Uptime:    formatUptime(uptime),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
		StartTime: h.startTime.UTC().Format(time.RFC3339),
		System: SystemMetrics{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			MemAllocMB:   m.Alloc / bytesToMB,
			MemTotalMB:   m.TotalAlloc / bytesToMB,
			NumGC:        m.NumGC,
		},
		API: map[string]interface{}{
			"version": "1.0.0",
			"mcp": map[string]interface{}{
				"enabled": true,
				"url":     "https://mcp.musicalaideas.com/mcp",
			},
		},
	}

	c.JSON(http.StatusOK, metrics)
}
