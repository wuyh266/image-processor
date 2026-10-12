package process

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"image-processor/internal/stats"
)

func newTestHandler() http.HandlerFunc {
	st := stats.New()
	svc := NewService(2, 10<<20, false, st)
	return NewResizeHandler(svc, 10*time.Second, 11<<20, st)
}

func TestResize_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", ct, "application/json; charset=utf-8")
	}

	var got ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error != "method not allowed" {
		t.Fatalf("error = %q, want %q", got.Error, "method not allowed")
	}
}

func TestResize_NoFile(t *testing.T) {
	handler := newTestHandler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("width", "100")
	writer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resize", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var got ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error != "file is required" {
		t.Fatalf("error = %q, want %q", got.Error, "file is required")
	}
}

func TestResize_InvalidWidth(t *testing.T) {
	handler := newTestHandler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.jpg"`)
	h.Set("Content-Type", "image/jpeg")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("fake-image-data"))
	writer.WriteField("width", "abc")
	writer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resize", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	var got ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error != "invalid width" {
		t.Fatalf("error = %q, want %q", got.Error, "invalid width")
	}
}

func TestResize_WidthOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		width string
	}{
		{"too small", "5"},
		{"too large", "5000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestHandler()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name="file"; filename="test.jpg"`)
			h.Set("Content-Type", "image/jpeg")
			part, _ := writer.CreatePart(h)
			part.Write([]byte("fake-image-data"))
			writer.WriteField("width", tt.width)
			writer.Close()

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/resize", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestResize_UnsupportedFormat(t *testing.T) {
	handler := newTestHandler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.gif"`)
	h.Set("Content-Type", "image/gif")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("GIF89a"))
	writer.WriteField("width", "100")
	writer.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resize", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
}
