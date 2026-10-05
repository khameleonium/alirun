package tui

import (
	"alirun/pkg/initsys"
	"context"
	"strings"
	"testing"
	"time"
)

func TestSortServices(t *testing.T) {
	now := time.Now()
	services := []initsys.ServiceInfo{
		{Name: "charlie", Status: initsys.StatusInactive, MemoryBytes: 200 * 1024 * 1024, CPUUsageNSec: 5000000000, ActiveSince: now.Add(-10 * time.Minute)},
		{Name: "alpha", Status: initsys.StatusActive, MemoryBytes: 50 * 1024 * 1024, CPUUsageNSec: 20000000000, ActiveSince: now.Add(-60 * time.Minute)},
		{Name: "bravo", Status: initsys.StatusFailed, MemoryBytes: 500 * 1024 * 1024, CPUUsageNSec: 1000000000, ActiveSince: now.Add(-30 * time.Minute)},
	}

	// 1. Sort by Name Asc
	SortServices(services, SortByName, SortAsc)
	if services[0].Name != "alpha" || services[1].Name != "bravo" || services[2].Name != "charlie" {
		t.Errorf("SortByName Asc failed: %v, %v, %v", services[0].Name, services[1].Name, services[2].Name)
	}

	// 2. Sort by Name Desc
	SortServices(services, SortByName, SortDesc)
	if services[0].Name != "charlie" || services[1].Name != "bravo" || services[2].Name != "alpha" {
		t.Errorf("SortByName Desc failed: %v, %v, %v", services[0].Name, services[1].Name, services[2].Name)
	}

	// 3. Sort by Status Asc (Active, Failed, Inactive)
	SortServices(services, SortByStatus, SortAsc)
	if services[0].Status != initsys.StatusActive || services[1].Status != initsys.StatusFailed || services[2].Status != initsys.StatusInactive {
		t.Errorf("SortByStatus failed")
	}

	// 4. Sort by Memory Desc
	SortServices(services, SortByMemory, SortDesc)
	if services[0].Name != "bravo" || services[1].Name != "charlie" || services[2].Name != "alpha" {
		t.Errorf("SortByMemory Desc failed: %v, %v, %v", services[0].Name, services[1].Name, services[2].Name)
	}

	// 5. Sort by CPU Desc
	SortServices(services, SortByCPU, SortDesc)
	if services[0].Name != "alpha" || services[1].Name != "charlie" || services[2].Name != "bravo" {
		t.Errorf("SortByCPU Desc failed: %v, %v, %v", services[0].Name, services[1].Name, services[2].Name)
	}

	// 6. Sort by Uptime Desc (only active service has uptime)
	SortServices(services, SortByUptime, SortDesc)
	if services[0].Name != "alpha" {
		t.Errorf("SortByUptime Desc failed, expected alpha first, got %s", services[0].Name)
	}
}

func TestFormatters(t *testing.T) {
	if FormatCPU(0) != "0.0s" {
		t.Errorf("expected 0.0s, got %s", FormatCPU(0))
	}
	if FormatCPU(45000000000) != "45.0s" {
		t.Errorf("expected 45.0s, got %s", FormatCPU(45000000000))
	}

	if FormatRAM(0) != "-" {
		t.Errorf("expected -, got %s", FormatRAM(0))
	}
	if FormatRAM(10*1024*1024) != "10.0 MB" {
		t.Errorf("expected 10.0 MB, got %s", FormatRAM(10*1024*1024))
	}

	if FormatPID(0) != "-" || FormatPID(1234) != "1234" {
		t.Errorf("unexpected FormatPID result")
	}
}

type dummyManager struct{}

func (d *dummyManager) Name() string                                                             { return "dummy" }
func (d *dummyManager) IsAvailable() bool                                                        { return true }
func (d *dummyManager) ListServices(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	return []initsys.ServiceInfo{
		{Name: "svc1.service", Status: initsys.StatusActive, SubState: "running", MemoryBytes: 100 * 1024 * 1024, CPUUsageNSec: 5000000000, PID: 1234},
		{Name: "svc2.service", Status: initsys.StatusFailed, SubState: "failed", MemoryBytes: 50 * 1024 * 1024, CPUUsageNSec: 1000000000, PID: 5678},
	}, nil
}
func (d *dummyManager) GetStatus(ctx context.Context, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	return &initsys.ServiceInfo{Name: name, Status: initsys.StatusActive}, nil
}
func (d *dummyManager) Start(ctx context.Context, name string, sType initsys.ServiceType) error   { return nil }
func (d *dummyManager) Stop(ctx context.Context, name string, sType initsys.ServiceType) error    { return nil }
func (d *dummyManager) Restart(ctx context.Context, name string, sType initsys.ServiceType) error { return nil }
func (d *dummyManager) Enable(ctx context.Context, name string, sType initsys.ServiceType) error  { return nil }
func (d *dummyManager) Disable(ctx context.Context, name string, sType initsys.ServiceType) error { return nil }
func (d *dummyManager) GenerateConfig(cfg initsys.ServiceConfig) (string, error)                  { return "", nil }
func (d *dummyManager) GetConfigPath(name string, sType initsys.ServiceType) string               { return "/mock" }
func (d *dummyManager) InstallService(ctx context.Context, cfg initsys.ServiceConfig, content string, enableNow bool) (string, error) {
	return "/mock", nil
}
func (d *dummyManager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	return nil
}
func (d *dummyManager) StreamLogs(ctx context.Context, name string, sType initsys.ServiceType, lines int, follow bool) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}

func TestViewModeToggleNoPanic(t *testing.T) {
	m := NewModel(&dummyManager{}, initsys.TypeSystem)
	m.width = 120
	m.height = 40
	m.rawServices = []initsys.ServiceInfo{
		{Name: "alpha.service", Status: initsys.StatusActive, SubState: "running", MemoryBytes: 200 * 1024 * 1024, CPUUsageNSec: 5000000000, PID: 123},
		{Name: "beta.service", Status: initsys.StatusFailed, SubState: "failed", MemoryBytes: 100 * 1024 * 1024, CPUUsageNSec: 1000000000, PID: 456},
	}
	m.recalcLayout()
	m.applyFilter()

	// Switch to Detailed
	m.viewMode = TableViewDetailed
	m.recalcLayout()
	m.applyFilter()
	_ = m.View()

	// Switch back to Compact (this was the exact sequence that previously panicked!)
	m.viewMode = TableViewCompact
	m.recalcLayout()
	m.applyFilter()
	_ = m.View()

	// Switch to Detailed again
	m.viewMode = TableViewDetailed
	m.recalcLayout()
	m.applyFilter()
	_ = m.View()
}

func TestFooterHintsAlwaysVisible(t *testing.T) {
	m := NewModel(&dummyManager{}, initsys.TypeSystem)
	m.width = 120
	m.height = 40
	m.statusMessage = "Sorted by CPU (▼)"

	footer := m.renderFooter()

	if !strings.Contains(footer, "Sorted by CPU (▼)") {
		t.Errorf("footer should contain the status message")
	}
	if !strings.Contains(footer, "[V]") || !strings.Contains(footer, "[O/P]") || !strings.Contains(footer, "[S]") {
		t.Errorf("footer must always contain key navigation hints, got: %s", footer)
	}
}
