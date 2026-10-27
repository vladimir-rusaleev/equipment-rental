package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"rent/internal/metrics"
	"rent/internal/platform"
	pb "rent/pkg/generated"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type backend interface {
	Authenticate(context.Context, string) (*pb.User, error)
	Call(context.Context, string, string, url.Values, json.RawMessage) (proto.Message, error)
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
	measured := metrics.HTTP(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		measured.ServeHTTP(w, r.WithContext(platform.InternalContext(ctx)))
	})
}

func (c *controller) handle(w http.ResponseWriter, r *http.Request) {
	ctx, user, err := c.authorize(r)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if r.Pattern == "GET /api/v1/me" {
		respond(w, user, nil)
		return
	}
	var raw json.RawMessage
	if r.Method != "GET" {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		raw, err = io.ReadAll(r.Body)
		if err != nil {
			respond(w, nil, status.Error(codes.InvalidArgument, "request body too large or unreadable"))
			return
		}
	}
	response, err := c.backend.Call(ctx, r.Pattern, r.PathValue("id"), r.URL.Query(), raw)
	respond(w, response, err)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"service": "rent-gateway",
		"links":   map[string]string{"frontend": "/", "catalog": "/api/v1/models", "health": "/healthz", "register": "/api/v1/auth/register", "login": "/api/v1/auth/login"},
	})
}
