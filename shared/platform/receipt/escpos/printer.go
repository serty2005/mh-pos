package escpos

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	PrinterTypeTCP = "tcp"
	PrinterTypeUSB = "usb"
)

// PrinterConfig описывает raw ESC/POS-принтер без routing/orchestration.
// Codepage: "" или "cp437" = CP437 (default, для English/European принтеров);
// "cp866" = PC866 Cyrillic (для русскоязычных развёртываний).
type PrinterConfig struct {
	Type             string `json:"type"`
	Address          string `json:"address"`
	Port             int    `json:"port"`
	CPL              int    `json:"cpl"`
	PrinterClass     string `json:"printer_class"`
	Codepage         string `json:"codepage"`
	RasterOnly       bool   `json:"raster_only"`
	PaperCutType     string `json:"paper_cut_type"`
	ConnectTimeoutMs int    `json:"connect_timeout_ms"`
	WriteTimeoutMs   int    `json:"write_timeout_ms"`
}

// RenderOptions возвращает настройки рендера, зависящие от принтера.
func (c PrinterConfig) RenderOptions() RenderOptions {
	return RenderOptions{CPL: c.CPL, RasterOnly: c.RasterOnly, PaperCutType: c.PaperCutType, Codepage: c.Codepage}
}

// WriteRaw пишет готовые ESC/POS-байты в TCP или USB raw destination.
func WriteRaw(ctx context.Context, cfg PrinterConfig, payload []byte) error {
	w, err := Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer w.Close()
	if deadlineWriter, ok := w.(interface{ SetWriteDeadline(time.Time) error }); ok {
		timeout := timeoutDuration(cfg.WriteTimeoutMs)
		if timeout > 0 {
			_ = deadlineWriter.SetWriteDeadline(time.Now().Add(timeout))
		}
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("printer write: %w", err)
	}
	return nil
}

// Open открывает raw writer к принтеру. Для USB address является путем вида \\.\USB001.
func Open(ctx context.Context, cfg PrinterConfig) (io.WriteCloser, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case PrinterTypeTCP:
		return openTCP(ctx, cfg)
	case PrinterTypeUSB:
		if strings.TrimSpace(cfg.Address) == "" {
			return nil, fmt.Errorf("printer usb address is required")
		}
		f, err := os.OpenFile(cfg.Address, os.O_WRONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("open usb printer %s: %w", cfg.Address, err)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("unsupported printer type %q", cfg.Type)
	}
}

func openTCP(ctx context.Context, cfg PrinterConfig) (io.WriteCloser, error) {
	if strings.TrimSpace(cfg.Address) == "" {
		return nil, fmt.Errorf("printer tcp address is required")
	}
	port := cfg.Port
	if port == 0 {
		port = 9100
	}
	addr := net.JoinHostPort(cfg.Address, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: timeoutDuration(cfg.ConnectTimeoutMs)}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect tcp printer %s: %w", addr, err)
	}
	return conn, nil
}

func timeoutDuration(ms int) time.Duration {
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
