package tui

import (
	"alirun/pkg/initsys"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TableViewMode represents the table density/detail level
type TableViewMode int

const (
	TableViewCompact TableViewMode = iota // Simple (ST, SERVICE, STATE)
	TableViewDetailed                     // Detailed (ST, SERVICE, STATE, CPU, RAM, UPTIME, PID)
)

func (v TableViewMode) String() string {
	if v == TableViewDetailed {
		return "Detailed"
	}
	return "Compact"
}

// SortField indicates which property is used to sort services
type SortField int

const (
	SortByName SortField = iota
	SortByStatus
	SortByStartTime
	SortByUptime
	SortByCPU
	SortByMemory
)

func (s SortField) String() string {
	switch s {
	case SortByName:
		return "Name"
	case SortByStatus:
		return "Status"
	case SortByStartTime:
		return "Start"
	case SortByUptime:
		return "Uptime"
	case SortByCPU:
		return "CPU"
	case SortByMemory:
		return "RAM"
	default:
		return "Name"
	}
}

// SortDirection indicates ascending or descending order
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

func (d SortDirection) Arrow() string {
	if d == SortDesc {
		return "▼"
	}
	return "▲"
}

// SortServices sorts a slice of ServiceInfo in-place
func SortServices(services []initsys.ServiceInfo, field SortField, dir SortDirection) {
	sort.SliceStable(services, func(i, j int) bool {
		a, b := services[i], services[j]
		var less bool
		switch field {
		case SortByName:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortByStatus:
			statusWeight := func(st initsys.ServiceStatus) int {
				switch st {
				case initsys.StatusActive:
					return 0
				case initsys.StatusFailed:
					return 1
				case initsys.StatusInactive:
					return 2
				default:
					return 3
				}
			}
			wa, wb := statusWeight(a.Status), statusWeight(b.Status)
			if wa != wb {
				less = wa < wb
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case SortByStartTime:
			if a.ActiveSince.Equal(b.ActiveSince) {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			} else if a.ActiveSince.IsZero() {
				less = false
			} else if b.ActiveSince.IsZero() {
				less = true
			} else {
				less = a.ActiveSince.Before(b.ActiveSince)
			}
		case SortByUptime:
			uptime := func(s initsys.ServiceInfo) time.Duration {
				if s.Status == initsys.StatusActive && !s.ActiveSince.IsZero() {
					return time.Since(s.ActiveSince)
				}
				return 0
			}
			ua, ub := uptime(a), uptime(b)
			if ua != ub {
				less = ua < ub
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case SortByCPU:
			if a.CPUUsageNSec != b.CPUUsageNSec {
				less = a.CPUUsageNSec < b.CPUUsageNSec
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case SortByMemory:
			if a.MemoryBytes != b.MemoryBytes {
				less = a.MemoryBytes < b.MemoryBytes
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}

		if dir == SortDesc {
			return !less
		}
		return less
	})
}

// FormatCPU formats nanoseconds into a friendly time string e.g. "44.4s", "1m20s"
func FormatCPU(cpuNSec uint64) string {
	if cpuNSec == 0 {
		return "0.0s"
	}
	sec := float64(cpuNSec) / 1e9
	if sec < 60.0 {
		return fmt.Sprintf("%.1fs", sec)
	}
	min := int(sec) / 60
	remSec := int(sec) % 60
	if min < 60 {
		return fmt.Sprintf("%dm%02ds", min, remSec)
	}
	hrs := min / 60
	remMin := min % 60
	return fmt.Sprintf("%dh%02dm", hrs, remMin)
}

// FormatRAM formats bytes into MB or GB
func FormatRAM(bytes uint64) string {
	if bytes == 0 {
		return "-"
	}
	mb := float64(bytes) / (1024 * 1024)
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

// FormatUptime formats service running duration
func FormatUptime(since time.Time, status initsys.ServiceStatus) string {
	if status != initsys.StatusActive || since.IsZero() {
		return "-"
	}
	d := time.Since(since)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	return fmt.Sprintf("%dd%02dh", days, int(d.Hours())%24)
}

// FormatPID returns string representation of PID or "-"
func FormatPID(pid int) string {
	if pid <= 0 {
		return "-"
	}
	return strconv.Itoa(pid)
}
