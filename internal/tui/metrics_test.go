package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderProgressBar(t *testing.T) {
	bar := RenderProgressBar(50, 10, lipgloss.Color("#00E676"))
	if !strings.Contains(bar, "[") || !strings.Contains(bar, "]") {
		t.Errorf("expected brackets in progress bar, got: %s", bar)
	}
	if !strings.Contains(bar, "█") || !strings.Contains(bar, "░") {
		t.Errorf("expected filled and empty blocks in 50%% bar, got: %s", bar)
	}

	bar0 := RenderProgressBar(0, 10, lipgloss.Color("#00E676"))
	if !strings.Contains(bar0, "░") {
		t.Errorf("expected empty blocks in 0%% bar, got: %s", bar0)
	}

	bar100 := RenderProgressBar(100, 10, lipgloss.Color("#00E676"))
	if !strings.Contains(bar100, "█") {
		t.Errorf("expected filled blocks in 100%% bar, got: %s", bar100)
	}
}

func TestRenderSparkline(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80}
	spark := RenderSparkline(values, lipgloss.Color("#7C4DFF"))
	if spark == "" {
		t.Errorf("expected non-empty sparkline")
	}

	emptySpark := RenderSparkline(nil, lipgloss.Color("#7C4DFF"))
	if emptySpark != "" {
		t.Errorf("expected empty sparkline for nil values")
	}
}

func TestServiceMetrics(t *testing.T) {
	m := &ServiceMetrics{}
	t0 := time.Now()

	// Initial sample
	m.AddSample(1000000000, 10*1024*1024, t0)
	if m.LastRAMMB != 10.0 {
		t.Errorf("expected 10.0 MB RAM, got %f", m.LastRAMMB)
	}

	// Next sample 1 second later with 0.5s CPU consumed (50% CPU)
	t1 := t0.Add(1 * time.Second)
	m.AddSample(1500000000, 12*1024*1024, t1)

	if m.LastCPUPercent < 45 || m.LastCPUPercent > 55 {
		t.Errorf("expected around 50%% CPU, got %f", m.LastCPUPercent)
	}
	if len(m.CPUHistory) != 2 {
		t.Errorf("expected 2 history points, got %d", len(m.CPUHistory))
	}
	if len(m.RAMHistory) != 2 {
		t.Errorf("expected 2 history points, got %d", len(m.RAMHistory))
	}
}
