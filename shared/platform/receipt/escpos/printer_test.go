package escpos

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestWriteRawTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	readCh := make(chan []byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			readCh <- nil
			return
		}
		defer conn.Close()
		raw, _ := io.ReadAll(conn)
		readCh <- raw
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x1b, '@', 'o', 'k'}
	if err := WriteRaw(context.Background(), PrinterConfig{
		Type:             PrinterTypeTCP,
		Address:          host,
		Port:             port,
		ConnectTimeoutMs: 1000,
		WriteTimeoutMs:   1000,
	}, payload); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-readCh:
		if string(got) != string(payload) {
			t.Fatalf("payload mismatch: % x", got)
		}
	case <-time.After(time.Second):
		t.Fatal("tcp listener did not receive payload")
	}
}

func TestWriteRawUSBPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "USB001")
	payload := []byte{0x1b, '@', 'u', 's', 'b'}
	if err := os.WriteFile(path, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := WriteRaw(context.Background(), PrinterConfig{Type: PrinterTypeUSB, Address: path}, payload); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch: % x", got)
	}
}
