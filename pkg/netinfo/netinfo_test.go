package netinfo

import (
	"os"
	"testing"
)

func TestParseHexAddressIPv4(t *testing.T) {
	// 0100007F:0050 -> 127.0.0.1:80
	ip, port, err := parseHexAddress("0100007F:0050")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "127.0.0.1" || port != 80 {
		t.Errorf("expected 127.0.0.1:80, got %s:%d", ip, port)
	}

	// 00000000:1F90 -> 0.0.0.0:8080 (0x1F90 = 8080)
	ip, port, err = parseHexAddress("00000000:1F90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "0.0.0.0" || port != 8080 {
		t.Errorf("expected 0.0.0.0:8080, got %s:%d", ip, port)
	}
}

func TestGetListeningPortsForPID(t *testing.T) {
	pid := os.Getpid()
	// Test on self
	ports := GetListeningPortsForPID(pid)
	t.Logf("Self listening ports: %v", ports)
}
