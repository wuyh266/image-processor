package process

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"image-processor/internal/stats"
)

func TestResize_RequestBodyLimit(t *testing.T) {
	data := requestLimitPNG(t)
	baseRequest := resizeImageRequest(t, data, "image/png", "20")
	requestSize := baseRequest.ContentLength
	baseRequest.Body.Close()

	tests := []struct {
		name   string
		budget int64
		status int
	}{
		{name: "below_limit", budget: requestSize + 1, status: http.StatusOK},
		{name: "at_limit", budget: requestSize, status: http.StatusOK},
		{name: "over_limit", budget: requestSize - 1, status: http.StatusRequestEntityTooLarge},
	}
	for _, knownLength := range []bool{true, false} {
		lengthName := "known_length"
		if !knownLength {
			lengthName = "unknown_length"
		}
		for _, tt := range tests {
			t.Run(lengthName+"/"+tt.name, func(t *testing.T) {
				st := stats.New()
				svc := NewService(1, int64(len(data)), false, st)
				handler := NewResizeHandler(svc, time.Second, tt.budget, st)
				req := resizeImageRequest(t, data, "image/png", "20")
				if !knownLength {
					req.ContentLength = -1
				}
				body := &countingRequestBody{ReadCloser: req.Body}
				req.Body = body
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				if body.bytesRead > tt.budget+1 {
					t.Fatalf("read %d bytes with a %d-byte budget", body.bytesRead, tt.budget)
				}
				if tt.status != http.StatusOK {
					assertResizeRequestError(t, rr, st, tt.status, "request body too large")
					return
				}
				if rr.Code != http.StatusOK {
					t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
				}
				result, format, err := image.Decode(bytes.NewReader(rr.Body.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				if format != "png" || result.Bounds().Dx() != 20 || result.Bounds().Dy() != 10 {
					t.Fatalf("result = %s %v, want png 20x10", format, result.Bounds())
				}
				if snap := st.Snapshot(); snap.TotalRequests != 1 || snap.Success != 1 || snap.Errors != 0 {
					t.Fatalf("unexpected counters: %+v", snap)
				}
			})
		}
	}
}

func TestResize_RequestBodyLimitIncludesFormFields(t *testing.T) {
	data := requestLimitPNG(t)
	st := stats.New()
	svc := NewService(1, int64(len(data)), false, st)
	const budget = 1024
	handler := NewResizeHandler(svc, time.Second, budget, st)
	req := resizeImageRequest(t, data, "image/png", strings.Repeat("1", 4096))
	req.ContentLength = -1
	body := &countingRequestBody{ReadCloser: req.Body}
	req.Body = body
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assertResizeRequestError(t, rr, st, http.StatusRequestEntityTooLarge, "request body too large")
	if body.bytesRead > budget+1 {
		t.Fatalf("read %d bytes with a %d-byte budget", body.bytesRead, budget)
	}
}

func TestResize_MalformedMultipartWithinBodyLimit(t *testing.T) {
	st := stats.New()
	svc := NewService(1, 128, false, st)
	handler := NewResizeHandler(svc, time.Second, 1024, st)
	req := httptest.NewRequest(http.MethodPost, "/resize", strings.NewReader("--test\r\nContent-Disposition"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assertResizeRequestError(t, rr, st, http.StatusBadRequest, "invalid multipart form")
}

func requestLimitPNG(t *testing.T) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func assertResizeRequestError(t *testing.T, rr *httptest.ResponseRecorder, st *stats.Stats, status int, message string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, body = %s, want %d", rr.Code, rr.Body.String(), status)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q, want JSON", ct)
	}
	var result ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Error != message {
		t.Fatalf("error = %q, want %q", result.Error, message)
	}
	if snap := st.Snapshot(); snap.TotalRequests != 1 || snap.Success != 0 || snap.Errors != 1 {
		t.Fatalf("unexpected counters: %+v", snap)
	}
}

type countingRequestBody struct {
	io.ReadCloser
	bytesRead int64
}

func (b *countingRequestBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.bytesRead += int64(n)
	return n, err
}
