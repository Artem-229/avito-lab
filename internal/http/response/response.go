package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Artem-229/avito-lab/internal/http/logcontext"
)

func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	WriteBody(ctx, w, "application/json", status, v)
}

func WriteBody(ctx context.Context, w http.ResponseWriter, contentType string, status int, v any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logcontext.Error(ctx, "write response body", "err", err)
	}
}
