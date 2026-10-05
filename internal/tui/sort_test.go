package tui

import (
	"alirun/pkg/initsys"
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
