package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var requestDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestMetricKey struct {
	method string
	route  string
	status int
}

type requestHistogram struct {
	count   uint64
	sum     float64
	buckets [11]uint64
}

type requestMetrics struct {
	mu                  sync.Mutex
	requests            map[requestMetricKey]uint64
	durations           map[struct{ method, route string }]*requestHistogram
	evaluations         map[string]uint64
	evaluationDurations map[string]*requestHistogram
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{requests: make(map[requestMetricKey]uint64), durations: make(map[struct{ method, route string }]*requestHistogram), evaluations: make(map[string]uint64), evaluationDurations: make(map[string]*requestHistogram)}
}

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *metricsResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (s *Server) measureRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		started := time.Now()
		recorder := &metricsResponseWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		s.metrics.observeHTTP(r.Method, metricRouteGroup(r.URL.Path), status, time.Since(started).Seconds())
	})
}

func metricRouteGroup(path string) string {
	switch {
	case path == "/healthz" || path == "/readyz":
		return "health"
	case strings.HasPrefix(path, "/api/v1/auth/"):
		return "auth"
	case strings.HasPrefix(path, "/api/v1/engine/"):
		return "engine"
	case strings.HasPrefix(path, "/api/v1/rulesets"):
		return "rulesets"
	case strings.HasPrefix(path, "/api/v1/projects"):
		return "projects"
	case strings.HasPrefix(path, "/api/v1/organizations"):
		return "organizations"
	case strings.HasPrefix(path, "/api/v1/platform/"):
		return "platform"
	case strings.HasPrefix(path, "/api/v1/"):
		return "api_other"
	default:
		return "web"
	}
}

func (m *requestMetrics) observeHTTP(method, route string, status int, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[requestMetricKey{method: method, route: route, status: status}]++
	key := struct{ method, route string }{method, route}
	h := m.durations[key]
	if h == nil {
		h = &requestHistogram{}
		m.durations[key] = h
	}
	observeDuration(h, seconds)
}

func observeDuration(h *requestHistogram, seconds float64) {
	h.count++
	h.sum += seconds
	for i, bound := range requestDurationBuckets {
		if seconds <= bound {
			h.buckets[i]++
		}
	}
}

func (m *requestMetrics) observeEvaluation(result string, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evaluations[result]++
	h := m.evaluationDurations[result]
	if h == nil {
		h = &requestHistogram{}
		m.evaluationDurations[result] = h
	}
	observeDuration(h, seconds)
}

func metricEscape(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(value)
}

func (s *Server) metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		m := s.metrics
		m.mu.Lock()
		requestKeys := make([]requestMetricKey, 0, len(m.requests))
		for key := range m.requests {
			requestKeys = append(requestKeys, key)
		}
		sort.Slice(requestKeys, func(i, j int) bool {
			a, b := requestKeys[i], requestKeys[j]
			if a.route != b.route {
				return a.route < b.route
			}
			if a.method != b.method {
				return a.method < b.method
			}
			return a.status < b.status
		})
		fmt.Fprintln(w, "# HELP snakerule_http_requests_total Total HTTP requests by bounded route group and status.")
		fmt.Fprintln(w, "# TYPE snakerule_http_requests_total counter")
		for _, key := range requestKeys {
			fmt.Fprintf(w, "snakerule_http_requests_total{method=%q,route_group=%q,status_code=%q} %d\n", metricEscape(key.method), metricEscape(key.route), strconv.Itoa(key.status), m.requests[key])
		}
		histogramKeys := make([]struct{ method, route string }, 0, len(m.durations))
		for key := range m.durations {
			histogramKeys = append(histogramKeys, key)
		}
		sort.Slice(histogramKeys, func(i, j int) bool {
			if histogramKeys[i].route != histogramKeys[j].route {
				return histogramKeys[i].route < histogramKeys[j].route
			}
			return histogramKeys[i].method < histogramKeys[j].method
		})
		fmt.Fprintln(w, "# HELP snakerule_http_request_duration_seconds HTTP request duration in seconds.")
		fmt.Fprintln(w, "# TYPE snakerule_http_request_duration_seconds histogram")
		for _, key := range histogramKeys {
			h := m.durations[key]
			for i, bound := range requestDurationBuckets {
				fmt.Fprintf(w, "snakerule_http_request_duration_seconds_bucket{method=%q,route_group=%q,le=%q} %d\n", metricEscape(key.method), metricEscape(key.route), strconv.FormatFloat(bound, 'f', -1, 64), h.buckets[i])
			}
			fmt.Fprintf(w, "snakerule_http_request_duration_seconds_bucket{method=%q,route_group=%q,le=\"+Inf\"} %d\n", metricEscape(key.method), metricEscape(key.route), h.count)
			fmt.Fprintf(w, "snakerule_http_request_duration_seconds_sum{method=%q,route_group=%q} %g\n", metricEscape(key.method), metricEscape(key.route), h.sum)
			fmt.Fprintf(w, "snakerule_http_request_duration_seconds_count{method=%q,route_group=%q} %d\n", metricEscape(key.method), metricEscape(key.route), h.count)
		}
		fmt.Fprintln(w, "# HELP snakerule_rule_evaluations_total Rule evaluation outcomes.")
		fmt.Fprintln(w, "# TYPE snakerule_rule_evaluations_total counter")
		for result, count := range m.evaluations {
			fmt.Fprintf(w, "snakerule_rule_evaluations_total{result=%q} %d\n", metricEscape(result), count)
		}
		fmt.Fprintln(w, "# HELP snakerule_rule_evaluation_duration_seconds Rule evaluation duration in seconds.")
		fmt.Fprintln(w, "# TYPE snakerule_rule_evaluation_duration_seconds histogram")
		for result, h := range m.evaluationDurations {
			for i, bound := range requestDurationBuckets {
				fmt.Fprintf(w, "snakerule_rule_evaluation_duration_seconds_bucket{result=%q,le=%q} %d\n", metricEscape(result), strconv.FormatFloat(bound, 'f', -1, 64), h.buckets[i])
			}
			fmt.Fprintf(w, "snakerule_rule_evaluation_duration_seconds_bucket{result=%q,le=\"+Inf\"} %d\n", metricEscape(result), h.count)
			fmt.Fprintf(w, "snakerule_rule_evaluation_duration_seconds_sum{result=%q} %g\n", metricEscape(result), h.sum)
			fmt.Fprintf(w, "snakerule_rule_evaluation_duration_seconds_count{result=%q} %d\n", metricEscape(result), h.count)
		}
		m.mu.Unlock()
		if s.database != nil {
			stats := s.database.Stats()
			fmt.Fprintln(w, "# HELP snakerule_database_connections Current PostgreSQL connections in the pool.")
			fmt.Fprintln(w, "# TYPE snakerule_database_connections gauge")
			fmt.Fprintf(w, "snakerule_database_connections{state=\"open\"} %d\n", stats.OpenConnections)
			fmt.Fprintf(w, "snakerule_database_connections{state=\"in_use\"} %d\n", stats.InUse)
			fmt.Fprintf(w, "snakerule_database_connections{state=\"idle\"} %d\n", stats.Idle)
			fmt.Fprintf(w, "snakerule_database_connections{state=\"max_open\"} %d\n", stats.MaxOpenConnections)
			ctx, cancel := context.WithTimeout(r.Context(), time.Second)
			err := s.database.PingContext(ctx)
			cancel()
			up := 0
			if err == nil {
				up = 1
			}
			fmt.Fprintln(w, "# HELP snakerule_database_up Whether PostgreSQL responds to a ping.")
			fmt.Fprintln(w, "# TYPE snakerule_database_up gauge")
			fmt.Fprintf(w, "snakerule_database_up %d\n", up)
		}
		if s.invocationLogs != nil {
			cleanup := s.invocationLogs.CleanupStatus()
			fmt.Fprintln(w, "# HELP snakerule_invocation_cleanup_last_run_timestamp_seconds Unix timestamp of the last retention cleanup attempt.")
			fmt.Fprintln(w, "# TYPE snakerule_invocation_cleanup_last_run_timestamp_seconds gauge")
			lastRun := float64(0)
			if !cleanup.LastRun.IsZero() {
				lastRun = float64(cleanup.LastRun.UnixNano()) / 1e9
			}
			fmt.Fprintf(w, "snakerule_invocation_cleanup_last_run_timestamp_seconds %g\n", lastRun)
			fmt.Fprintln(w, "# HELP snakerule_invocation_cleanup_last_deleted Number of records deleted in the last retention cleanup.")
			fmt.Fprintln(w, "# TYPE snakerule_invocation_cleanup_last_deleted gauge")
			fmt.Fprintf(w, "snakerule_invocation_cleanup_last_deleted %d\n", cleanup.LastDeleted)
			fmt.Fprintln(w, "# HELP snakerule_invocation_cleanup_total_deleted Total records deleted since this process started.")
			fmt.Fprintln(w, "# TYPE snakerule_invocation_cleanup_total_deleted counter")
			fmt.Fprintf(w, "snakerule_invocation_cleanup_total_deleted %d\n", cleanup.TotalDeleted)
			errorState := 0
			if cleanup.LastError != "" {
				errorState = 1
			}
			fmt.Fprintln(w, "# HELP snakerule_invocation_cleanup_last_run_error Whether the last retention cleanup run failed.")
			fmt.Fprintln(w, "# TYPE snakerule_invocation_cleanup_last_run_error gauge")
			fmt.Fprintf(w, "snakerule_invocation_cleanup_last_run_error %d\n", errorState)
		}
	})
}
