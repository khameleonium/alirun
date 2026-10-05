package netinfo

import (
	"net"
	"os"
	"strconv"
	"strings"
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
	// Start a local test listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind test port: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	pid := os.Getpid()

	ports := GetListeningPortsForPID(pid)
	t.Logf("Self listening ports: %v (looking for :%d)", ports, addr.Port)

	found := false
	expected := strconv.Itoa(addr.Port)
	for _, p := range ports {
		if strings.Contains(p, expected) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected port %d in %v", addr.Port, ports)
	}
}

func TestFormatSummary(t *testing.T) {
	s1 := NetSummary{
		Listening: []string{":8080 (tcp)"},
	}
	if s1.FormatSummary() != ":8080 (tcp)" {
		t.Errorf("unexpected: %s", s1.FormatSummary())
	}

	s2 := NetSummary{
		OutboundRemotes: []string{"149.154.167.50:443", "149.154.167.35:443"},
		OutboundCount:   3,
	}
	expected := "None listening (3 outbound -> 149.154.167.50:443, 149.154.167.35:443)"
	if s2.FormatSummary() != expected {
		t.Errorf("expected %q, got %q", expected, s2.FormatSummary())
	}

	s3 := NetSummary{}
	if s3.FormatSummary() != "None (no active connections)" {
		t.Errorf("expected 'None (no active connections)', got %q", s3.FormatSummary())
	}
}
