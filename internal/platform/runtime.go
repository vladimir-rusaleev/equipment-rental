package platform

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func InternalContext(ctx context.Context) context.Context {
	m, _ := metadata.FromOutgoingContext(ctx)
	m = m.Copy()
	m.Set("x-service-key", os.Getenv("SERVICE_KEY"))
	return metadata.NewOutgoingContext(ctx, m)
}

func ActorContext(ctx context.Context, id, role string) context.Context {
	ctx = InternalContext(ctx)
	m, _ := metadata.FromOutgoingContext(ctx)
	m = m.Copy()
	m.Set("x-user-id", id)
	m.Set("x-user-role", role)
	return metadata.NewOutgoingContext(ctx, m)
}

type actorKey struct{}
type identity struct{ ID, Role string }

func WithActor(ctx context.Context, id, role string) context.Context {
	return context.WithValue(ctx, actorKey{}, identity{ID: id, Role: role})
}

func Actor(ctx context.Context) (string, string) {
	v, _ := ctx.Value(actorKey{}).(identity)
	return v.ID, v.Role
}

func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func Context() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func Serve(ctx context.Context, addr string, register func(*grpc.Server)) error {
	key := os.Getenv("SERVICE_KEY")
	if len(key) < 16 {
		return errors.New("SERVICE_KEY must contain at least 16 characters")
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		m, _ := metadata.FromIncomingContext(ctx)
		v := m.Get("x-service-key")
		if len(v) != 1 || subtle.ConstantTimeCompare([]byte(v[0]), []byte(key)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}
		first := func(k string) string {
			values := m.Get(k)
			if len(values) == 1 {
				return values[0]
			}
			return ""
		}
		return next(WithActor(ctx, first("x-user-id"), first("x-user-role")), req)
	}))
	register(s)
	h := health.NewServer()
	healthpb.RegisterHealthServer(s, h)
	h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() { s.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.Stop()
		}
	}()
	return s.Serve(l)
}
