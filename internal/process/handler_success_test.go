package process

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"image-processor/internal/stats"
)

func TestResize_RealImage(t *testing.T) {
	for _, format := range []string{"jpeg", "png"} {
		for _, pooled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/pool=%t", format, pooled), func(t *testing.T) {
				src := image.NewRGBA(image.Rect(0, 0, 80, 40))
				for y := 0; y < 40; y++ {
					for x := 0; x < 80; x++ {
						src.SetRGBA(x, y, color.RGBA{R: uint8(x * 3), G: uint8(y * 6), B: 80, A: 255})
					}
				}
				var encoded bytes.Buffer
				var err error
				if format == "jpeg" {
					err = jpeg.Encode(&encoded, src, &jpeg.Options{Quality: 85})
				} else {
					err = png.Encode(&encoded, src)
				}
				if err != nil {
					t.Fatal(err)
				}

				st := stats.New()
				svc := NewService(2, 10<<20, pooled, st)
				handler := NewResizeHandler(svc, time.Second, 11<<20, st)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, resizeImageRequest(t, encoded.Bytes(), "image/"+format, "40"))
				if rr.Code != http.StatusOK {
					t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
				}
				if got := rr.Header().Get("Content-Type"); got != "image/"+format {
					t.Fatalf("content type = %q", got)
				}
				result, gotFormat, err := image.Decode(bytes.NewReader(rr.Body.Bytes()))
				if err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if gotFormat != format || result.Bounds().Dx() != 40 || result.Bounds().Dy() != 20 {
					t.Fatalf("result = %s %v, want %s 40x20", gotFormat, result.Bounds(), format)
				}
				if snapshot := st.Snapshot(); snapshot.TotalRequests != 1 || snapshot.Success != 1 || snapshot.Errors != 0 {
					t.Fatalf("unexpected request counters: %+v", snapshot)
				}
			})
		}
	}
}

func TestResize_CorruptImage(t *testing.T) {
	for _, pooled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pool=%t", pooled), func(t *testing.T) {
			st := stats.New()
			svc := NewService(2, 10<<20, pooled, st)
			handler := NewResizeHandler(svc, time.Second, 11<<20, st)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, resizeImageRequest(t, []byte("not an encoded image"), "image/png", "40"))
			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusUnprocessableEntity, rr.Body.String())
			}
			if snapshot := st.Snapshot(); snapshot.TotalRequests != 1 || snapshot.Success != 0 || snapshot.Errors != 1 {
				t.Fatalf("unexpected request counters: %+v", snapshot)
			}
		})
	}
}

func resizeImageRequest(t *testing.T, data []byte, contentType, width string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="sample"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("width", width); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/resize", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
