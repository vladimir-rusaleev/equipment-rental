package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rent_http_requests_total", Help: "Completed API requests by route and status.",
	}, []string{"method", "route", "code"})
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "rent_http_request_duration_seconds", Help: "API request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
	grpcRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rent_grpc_requests_total", Help: "Completed gRPC requests by method and status.",
	}, []string{"method", "code"})
	grpcDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "rent_grpc_request_duration_seconds", Help: "gRPC request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
	queueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rent_queue_pending", Help: "Pending events, refreshed by the worker once per cycle.",
	}, []string{"queue"})
	notificationAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rent_notification_attempts_total", Help: "Notification processing attempts, including retries.",
	}, []string{"stage", "result"})
)

func Handler() http.Handler { return promhttp.Handler() }

// HTTP records matched route patterns rather than URLs containing user or order IDs.
func HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		response := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		method := r.Method
		switch method {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		default:
			method = "OTHER"
		}
		code := response.code
		if code == 0 {
			code = http.StatusOK
		}
		httpRequests.WithLabelValues(method, route, strconv.Itoa(code)).Inc()
		httpDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (w *responseWriter) WriteHeader(code int) {
	if w.code != 0 {
		return
	}
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *responseWriter) Write(data []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func GRPC(ctx context.Context, request any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
	start := time.Now()
	response, err := next(ctx, request)
	grpcRequests.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
	grpcDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
	return response, err
}

func SetQueueDepth(queue string, count int) { queueDepth.WithLabelValues(queue).Set(float64(count)) }
func NotificationAttempt(stage string, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	notificationAttempts.WithLabelValues(stage, result).Inc()
}
