package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"rent/internal/platform"

	"google.golang.org/protobuf/proto"
)

type backend interface {
	Call(context.Context, string, string, url.Values) (proto.Message, error)
}

type controller struct{ backend backend }

func New(backend backend) http.Handler {
	c := &controller{backend: backend}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1", index)
	mux.HandleFunc("GET /api/v1/{$}", index)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	for _, route := range routes {
		mux.HandleFunc(route, c.handle)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r.WithContext(platform.InternalContext(ctx)))
	})
}

func (c *controller) handle(w http.ResponseWriter, r *http.Request) {
	response, err := c.backend.Call(r.Context(), r.Pattern, r.PathValue("id"), r.URL.Query())
	respond(w, response, err)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"service": "rent-gateway",
		"links":   map[string]string{"frontend": "/", "catalog": "/api/v1/models", "health": "/healthz"},
	})
}
