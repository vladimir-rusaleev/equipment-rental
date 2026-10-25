package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func respond(w http.ResponseWriter, v proto.Message, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		code := http.StatusInternalServerError
		switch status.Code(err) {
		case codes.InvalidArgument:
			code = 400
		case codes.Unauthenticated:
			code = 401
		case codes.PermissionDenied:
			code = 403
		case codes.NotFound:
			code = 404
		case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted:
			code = 409
		case codes.Unavailable:
			code = 503
		case codes.DeadlineExceeded:
			code = 504
		}
		message := status.Convert(err).Message()
		if code == 500 {
			slog.Error("request failed", "error", err)
			message = "internal error"
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	b, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(v)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"encoding error"}`))
		return
	}
	_, _ = w.Write(b)
}
