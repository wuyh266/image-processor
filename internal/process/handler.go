package process

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"image-processor/internal/stats"
)

func NewResizeHandler(svc *Service, timeout time.Duration, st *stats.Stats) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st.RecordRequest()

		if r.Method != http.MethodPost {
			st.RecordError()
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if err := r.ParseMultipartForm(32 << 20); err != nil {
			st.RecordError()
			writeJSONError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			st.RecordError()
			writeJSONError(w, http.StatusBadRequest, "file is required")
			return
		}
		defer file.Close()

		contentType := header.Header.Get("Content-Type")
		format, ok := detectFormat(contentType)
		if !ok {
			st.RecordError()
			writeJSONError(w, http.StatusUnsupportedMediaType, "unsupported image format, only jpeg and png are supported")
			return
		}

		widthStr := r.FormValue("width")
		if widthStr == "" {
			st.RecordError()
			writeJSONError(w, http.StatusBadRequest, "width is required")
			return
		}

		width, err := strconv.Atoi(widthStr)
		if err != nil {
			st.RecordError()
			writeJSONError(w, http.StatusBadRequest, "invalid width")
			return
		}
		if width < 10 || width > 4096 {
			st.RecordError()
			writeJSONError(w, http.StatusBadRequest, "width must be between 10 and 4096")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		result, err := svc.Resize(ctx, file, width, format)
		if err != nil {
			st.RecordError()
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				writeJSONError(w, http.StatusGatewayTimeout, "process timeout")
			case errors.Is(err, context.Canceled):
				writeJSONError(w, http.StatusRequestTimeout, "request canceled")
			case errors.Is(err, ErrFileTooLarge):
				writeJSONError(w, http.StatusRequestEntityTooLarge, "file too large")
			case errors.Is(err, ErrDecode):
				writeJSONError(w, http.StatusUnprocessableEntity, "failed to decode image")
			default:
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		st.RecordSuccess()

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(result)
	}
}

func detectFormat(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return "jpeg", true
	case "image/png":
		return "png", true
	default:
		return "", false
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
