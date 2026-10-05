package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

var sparkRunes = []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

type ServiceMetrics struct {
	LastSampleTime time.Time
	LastCPUNSec    uint64
	CPUHistory     []float64 // Last 16 samples
	RAMHistory     []float64 // Last 16 samples in MB
	LastCPUPercent float64
	LastRAMMB      float64
}

func (m *ServiceMetrics) AddSample(cpuNSec uint64, ramBytes uint64, sampleTime time.Time) {
	ramMB := float64(ramBytes) / (1024 * 1024)
	m.LastRAMMB = ramMB

	if !m.LastSampleTime.IsZero() && sampleTime.After(m.LastSampleTime) && cpuNSec >= m.LastCPUNSec {
		dt := sampleTime.Sub(m.LastSampleTime).Seconds()
		if dt > 0.1 && dt < 10.0 {
			dcpu := float64(cpuNSec - m.LastCPUNSec) / 1e9
			cpuPct := (dcpu / dt) * 100.0
			if cpuPct < 0 {
				cpuPct = 0
			}
			m.LastCPUPercent = cpuPct
		}
	} else if m.LastSampleTime.IsZero() {
		m.LastCPUPercent = 0
	}

	m.LastCPUNSec = cpuNSec
	m.LastSampleTime = sampleTime

	// Append to CPUHistory
	m.CPUHistory = append(m.CPUHistory, m.LastCPUPercent)
	if len(m.CPUHistory) > 16 {
		m.CPUHistory = m.CPUHistory[len(m.CPUHistory)-16:]
	}

	// Append to RAMHistory
	m.RAMHistory = append(m.RAMHistory, ramMB)
	if len(m.RAMHistory) > 16 {
		m.RAMHistory = m.RAMHistory[len(m.RAMHistory)-16:]
	}
}

// RenderSparkline creates a Unicode sparkline ( ▂▃▄▅▆▇█) from a slice of floats
func RenderSparkline(values []float64, color lipgloss.Color) string {
	if len(values) == 0 {
		return ""
	}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	diff := maxVal - minVal
	var runes []rune
	for _, v := range values {
		idx := 0
		if diff > 0.0001 {
			norm := (v - minVal) / diff
			idx = int(math.Round(norm * float64(len(sparkRunes)-1)))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(sparkRunes) {
				idx = len(sparkRunes) - 1
			}
		}
		runes = append(runes, sparkRunes[idx])
	}

	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(string(runes))
}

// RenderProgressBar creates a visual gauge bar e.g. [██████░░░░]
func RenderProgressBar(pct float64, width int, barColor lipgloss.Color) string {
	if width <= 2 {
		width = 10
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	filledLen := int(math.Round((pct / 100.0) * float64(width)))
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	filledStr := strings.Repeat("█", filledLen)
	emptyStr := strings.Repeat("░", emptyLen)

	bar := lipgloss.NewStyle().Foreground(barColor).Render(filledStr) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#455A64")).Render(emptyStr)

	return fmt.Sprintf("[%s]", bar)
}
