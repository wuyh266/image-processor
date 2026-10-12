package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image-processor/internal/config"
)

func TestLoad_RequestBodyLimit(t *testing.T) {
	const validConfig = `server:
  addr: ":8080"
  read_header_timeout: "1s"
  read_timeout: "1s"
  write_timeout: "1s"
  idle_timeout: "1s"
  shutdown_timeout: "1s"
process:
  timeout: "1s"
  max_concurrent: 1
  max_file_size: 1024
  max_request_size: 4096
pool:
  enabled: false
  capacity: 1
`
	tests := []struct {
		name    string
		setting string
		want    int64
		wantErr bool
	}{
		{name: "valid", setting: "  max_request_size: 4096\n", want: 4096},
		{name: "custom", setting: "  max_request_size: 8192\n", want: 8192},
		{name: "missing", setting: "", wantErr: true},
		{name: "zero", setting: "  max_request_size: 0\n", wantErr: true},
		{name: "negative", setting: "  max_request_size: -1\n", wantErr: true},
		{name: "equal_to_file_limit", setting: "  max_request_size: 1024\n", wantErr: true},
		{name: "below_file_limit", setting: "  max_request_size: 512\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := strings.Replace(validConfig, "  max_request_size: 4096\n", tt.setting, 1)
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(path)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "process.max_request_size") {
					t.Fatalf("error = %v, want a max_request_size configuration error", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Process.MaxRequestSize != tt.want || cfg.Process.MaxFileSize != 1024 {
				t.Fatalf("limits = request %d / file %d, want request %d / file 1024", cfg.Process.MaxRequestSize, cfg.Process.MaxFileSize, tt.want)
			}
		})
	}
}
