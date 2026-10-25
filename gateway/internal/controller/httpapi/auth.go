package httpapi

import (
	"context"
	"net/http"
	"strings"

	"rent/internal/platform"
	pb "rent/pkg/generated"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *controller) authorize(r *http.Request) (context.Context, *pb.User, error) {
	ctx := r.Context()
	role := requiredRole(r.Pattern)
	if role == "public" {
		return ctx, nil, nil
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ctx, nil, status.Error(codes.Unauthenticated, "bearer token required")
	}
	user, err := c.backend.Authenticate(ctx, strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		return ctx, nil, err
	}
	if role == "admin" && user.Role != "admin" || role == "staff" && user.Role != "admin" && user.Role != "staff" {
		return ctx, nil, status.Error(codes.PermissionDenied, "insufficient permissions")
	}
	return platform.ActorContext(ctx, user.Id, user.Role), user, nil
}

func requiredRole(route string) string {
	switch route {
	case "POST /api/v1/auth/register", "POST /api/v1/auth/login", "GET /api/v1/models", "GET /api/v1/models/{id}", "GET /api/v1/models/{id}/calendar":
		return "public"
	case "POST /api/v1/models", "PUT /api/v1/models/{id}", "PATCH /api/v1/users/{id}/role":
		return "admin"
	case "GET /api/v1/models/{id}/units", "POST /api/v1/models/{id}/units", "PATCH /api/v1/units/{id}":
		return "staff"
	default:
		return "customer"
	}
}
