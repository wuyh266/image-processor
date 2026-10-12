package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type rawConfig struct {
	Server struct {
		Addr              string `yaml:"addr"`
		ReadHeaderTimeout string `yaml:"read_header_timeout"`
		ReadTimeout       string `yaml:"read_timeout"`
		WriteTimeout      string `yaml:"write_timeout"`
		IdleTimeout       string `yaml:"idle_timeout"`
		ShutdownTimeout   string `yaml:"shutdown_timeout"`
	} `yaml:"server"`
	Process struct {
		Timeout        string `yaml:"timeout"`
		MaxConcurrent  int    `yaml:"max_concurrent"`
		MaxFileSize    int64  `yaml:"max_file_size"`
		MaxRequestSize int64  `yaml:"max_request_size"`
	} `yaml:"process"`
	Pool struct {
		Enabled  bool `yaml:"enabled"`
		Capacity int  `yaml:"capacity"`
	} `yaml:"pool"`
}

type Config struct {
	Server  ServerConfig
	Process ProcessConfig
	Pool    PoolConfig
}

type ServerConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type ProcessConfig struct {
	Timeout        time.Duration
	MaxConcurrent  int
	MaxFileSize    int64
	MaxRequestSize int64
}

type PoolConfig struct {
	Enabled  bool
	Capacity int
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg, err := convert(raw)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func mustParseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return d, nil
}

func convert(raw rawConfig) (Config, error) {
	readHeaderTimeout, err := mustParseDuration(raw.Server.ReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := mustParseDuration(raw.Server.ReadTimeout)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := mustParseDuration(raw.Server.WriteTimeout)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := mustParseDuration(raw.Server.IdleTimeout)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := mustParseDuration(raw.Server.ShutdownTimeout)
	if err != nil {
		return Config{}, err
	}
	processTimeout, err := mustParseDuration(raw.Process.Timeout)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Server: ServerConfig{
			Addr:              raw.Server.Addr,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			ShutdownTimeout:   shutdownTimeout,
		},
		Process: ProcessConfig{
			Timeout:        processTimeout,
			MaxConcurrent:  raw.Process.MaxConcurrent,
			MaxFileSize:    raw.Process.MaxFileSize,
			MaxRequestSize: raw.Process.MaxRequestSize,
		},
		Pool: PoolConfig{
			Enabled:  raw.Pool.Enabled,
			Capacity: raw.Pool.Capacity,
		},
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if cfg.Server.Addr == "" {
		return fmt.Errorf("server.addr must not be empty")
	}
	if cfg.Server.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("server.read_header_timeout must be > 0")
	}
	if cfg.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server.read_timeout must be > 0")
	}
	if cfg.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server.write_timeout must be > 0")
	}
	if cfg.Server.IdleTimeout <= 0 {
		return fmt.Errorf("server.idle_timeout must be > 0")
	}
	if cfg.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server.shutdown_timeout must be > 0")
	}
	if cfg.Process.Timeout <= 0 {
		return fmt.Errorf("process.timeout must be > 0")
	}
	if cfg.Process.MaxConcurrent <= 0 {
		return fmt.Errorf("process.max_concurrent must be > 0")
	}
	if cfg.Process.MaxFileSize <= 0 {
		return fmt.Errorf("process.max_file_size must be > 0")
	}
	if cfg.Process.MaxRequestSize <= 0 {
		return fmt.Errorf("process.max_request_size must be > 0")
	}
	if cfg.Process.MaxRequestSize <= cfg.Process.MaxFileSize {
		return fmt.Errorf("process.max_request_size must be > process.max_file_size to allow multipart overhead")
	}
	if cfg.Pool.Capacity <= 0 {
		return fmt.Errorf("pool.capacity must be > 0")
	}
	return nil
}
