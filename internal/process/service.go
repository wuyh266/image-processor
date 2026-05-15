package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"

	"image-processor/internal/pool"
	"image-processor/internal/stats"
)

var (
	ErrFileTooLarge = errors.New("file too large")
	ErrDecode       = errors.New("decode failed")
)

type Service struct {
	sem         chan struct{}
	maxFileSize int64
	poolEnabled bool
	st          *stats.Stats
}

func NewService(maxConcurrent int, maxFileSize int64, poolEnabled bool, st *stats.Stats) *Service {
	return &Service{
		sem:         make(chan struct{}, maxConcurrent),
		maxFileSize: maxFileSize,
		poolEnabled: poolEnabled,
		st:          st,
	}
}

func (s *Service) Resize(ctx context.Context, file io.Reader, targetWidth int, format string) ([]byte, error) {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	bufSize := int(s.maxFileSize + 1)
	var imgData []byte
	if s.poolEnabled {
		buf := pool.GetBuf(bufSize)
		defer pool.PutBuf(bufSize, buf)
		lr := io.LimitReader(file, s.maxFileSize+1)
		n, err := io.ReadFull(lr, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("read file: %w", err)
		}
		if n > int(s.maxFileSize) {
			return nil, ErrFileTooLarge
		}
		imgData = buf[:n]
	} else {
		data, err := io.ReadAll(io.LimitReader(file, s.maxFileSize+1))
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		if int64(len(data)) > s.maxFileSize {
			return nil, ErrFileTooLarge
		}
		imgData = data
	}

	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", srcW, srcH)
	}

	newH := targetWidth * srcH / srcW
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var outBuf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&outBuf, dst)
	default:
		err = jpeg.Encode(&outBuf, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return outBuf.Bytes(), nil
}
