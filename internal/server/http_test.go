package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"image-processor/internal/config"
	"image-processor/internal/process"
	"image-processor/internal/stats"
)

func TestRouter_RequestBodyLimit(t *testing.T) {
	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="sample.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not an encoded image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("width", "20"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("metadata", strings.Repeat("x", 4096)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	for _, knownLength := range []bool{true, false} {
		name := "content_length"
		if !knownLength {
			name = "chunked"
		}
		t.Run(name, func(t *testing.T) {
			cfg, err := config.Load("../../configs/config.yaml")
			if err != nil {
				t.Fatal(err)
			}
			cfg.Process.MaxFileSize = 128
			cfg.Process.MaxRequestSize = 1024
			st := stats.New()
			srv := httptest.NewServer(NewRouter(cfg, st))
			t.Cleanup(srv.Close)
			client := srv.Client()
			client.Timeout = 5 * time.Second
			var body io.Reader = bytes.NewReader(upload.Bytes())
			if !knownLength {
				body = struct{ io.Reader }{body}
			}
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/resize", body)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413", resp.StatusCode)
			}
			var result process.ErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.Error != "request body too large" {
				t.Fatalf("error = %q, want request body too large", result.Error)
			}
			if snap := st.Snapshot(); snap.TotalRequests != 1 || snap.Success != 0 || snap.Errors != 1 {
				t.Fatalf("unexpected counters: %+v", snap)
			}
		})
	}
}
