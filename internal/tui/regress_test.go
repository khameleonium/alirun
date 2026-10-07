package tui

import (
	"alirun/pkg/editor"
	"alirun/pkg/initsys"
	"testing"
	"time"
)

func TestSortStartTimeDescKeepsNeverStartedLast(t *testing.T) {
	now := time.Now()
	services := []initsys.ServiceInfo{
		{Name: "never"},
		{Name: "old", Status: initsys.StatusActive, ActiveSince: now.Add(-time.Hour)},
		{Name: "new", Status: initsys.StatusActive, ActiveSince: now.Add(-time.Minute)},
	}
	SortServices(services, SortByStartTime, SortDesc)
	if services[0].Name != "new" || services[2].Name != "never" {
		t.Errorf("unexpected order: %s, %s, %s", services[0].Name, services[1].Name, services[2].Name)
	}
	SortServices(services, SortByStartTime, SortAsc)
	if services[0].Name != "old" || services[2].Name != "never" {
		t.Errorf("unexpected asc order: %s, %s, %s", services[0].Name, services[1].Name, services[2].Name)
	}
}

func TestEditorFinishedRunsOnSavedBeforeCleanup(t *testing.T) {
	m := NewModel(nil, initsys.TypeUser)
	var order []string
	session := &editor.EditSession{
		OnSaved: func() error { order = append(order, "saved"); return nil },
		Cleanup: func() { order = append(order, "cleanup") },
	}
	m.Update(editorFinishedMsg{session: session})
	if len(order) != 2 || order[0] != "saved" || order[1] != "cleanup" {
		t.Errorf("OnSaved must run before Cleanup, got %v", order)
	}
}

func TestStaleLogMessageIsIgnored(t *testing.T) {
	m := NewModel(nil, initsys.TypeUser)
	m.logGen = 2
	m.rawLogsLines = []string{"Loading logs for b..."}
	m.Update(logLineMsg{gen: 1, text: "line from service a"})
	if len(m.rawLogsLines) != 1 || m.rawLogsLines[0] != "Loading logs for b..." {
		t.Errorf("stale logs were applied: %v", m.rawLogsLines)
	}
	m.Update(logLineMsg{gen: 2, text: "line from service b"})
	if m.rawLogsLines[0] != "line from service b" {
		t.Errorf("current logs not applied: %v", m.rawLogsLines)
	}
}
