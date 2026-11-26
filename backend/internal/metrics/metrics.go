package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type routeStats struct {
	counts        map[string]uint64
	durationSum   float64
	durationCount uint64
}

type Collector struct {
	mu       sync.Mutex
	routes   map[string]*routeStats
	security map[string]uint64
}

func New() *Collector {
	return &Collector{routes: map[string]*routeStats{}, security: map[string]uint64{}}
}

var defaultCollector = New()

func Default() *Collector { return defaultCollector }

func (collector *Collector) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		collector.record(route, c.Request.Method, status, time.Since(start))
	}
}

func (collector *Collector) SecurityEvent(name string) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.security[name]++
}

func (collector *Collector) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(collector.render()))
	}
}

func (collector *Collector) record(route, method string, status int, latency time.Duration) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	stats := collector.routes[route]
	if stats == nil {
		stats = &routeStats{counts: map[string]uint64{}}
		collector.routes[route] = stats
	}
	key := method + "|" + statusClass(status)
	stats.counts[key]++
	stats.durationSum += latency.Seconds()
	stats.durationCount++
	switch status {
	case http.StatusUnauthorized:
		collector.security["unauthorized"]++
	case http.StatusForbidden:
		collector.security["forbidden"]++
	case http.StatusTooManyRequests:
		collector.security["rate_limited"]++
	}
	if status >= http.StatusInternalServerError {
		collector.security["server_errors"]++
	}
}

func (collector *Collector) render() string {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	routes := make([]string, 0, len(collector.routes))
	for route := range collector.routes {
		routes = append(routes, route)
	}
	sort.Strings(routes)

	var out strings.Builder
	out.WriteString("# HELP pharmaos_http_requests_total Total HTTP requests handled by route.\n")
	out.WriteString("# TYPE pharmaos_http_requests_total counter\n")
	for _, route := range routes {
		stats := collector.routes[route]
		keys := make([]string, 0, len(stats.counts))
		for key := range stats.counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts := strings.Split(key, "|")
			labels := fmt.Sprintf("route=%q,method=%q,status=%q", route, parts[0], parts[1])
			fmt.Fprintf(&out, "pharmaos_http_requests_total{%s} %d\n", labels, stats.counts[key])
		}
	}

	out.WriteString("# HELP pharmaos_http_request_duration_seconds HTTP request latency.\n")
	out.WriteString("# TYPE pharmaos_http_request_duration_seconds summary\n")
	for _, route := range routes {
		stats := collector.routes[route]
		if stats.durationCount == 0 {
			continue
		}
		labels := fmt.Sprintf("route=%q", route)
		fmt.Fprintf(&out, "pharmaos_http_request_duration_seconds_sum{%s} %.6f\n", labels, stats.durationSum)
		fmt.Fprintf(&out, "pharmaos_http_request_duration_seconds_count{%s} %d\n", labels, stats.durationCount)
	}

	out.WriteString("# HELP pharmaos_security_events_total Security-relevant HTTP outcomes.\n")
	out.WriteString("# TYPE pharmaos_security_events_total counter\n")
	events := make([]string, 0, len(collector.security))
	for event := range collector.security {
		events = append(events, event)
	}
	sort.Strings(events)
	for _, event := range events {
		fmt.Fprintf(&out, "pharmaos_security_events_total{event=%q} %s\n", event, strconv.FormatUint(collector.security[event], 10))
	}
	return out.String()
}

func statusClass(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
