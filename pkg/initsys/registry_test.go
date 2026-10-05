package initsys

import (
	"context"
	"testing"
)

type mockManager struct {
	name      string
	available bool
}

func (m *mockManager) Name() string                                                             { return m.name }
func (m *mockManager) IsAvailable() bool                                                        { return m.available }
func (m *mockManager) ListServices(ctx context.Context, sType ServiceType) ([]ServiceInfo, error) {
	return []ServiceInfo{{Name: "mock-service", Status: StatusActive}}, nil
}
func (m *mockManager) GetStatus(ctx context.Context, name string, sType ServiceType) (*ServiceInfo, error) {
	return &ServiceInfo{Name: name, Status: StatusActive}, nil
}
func (m *mockManager) Start(ctx context.Context, name string, sType ServiceType) error   { return nil }
func (m *mockManager) Stop(ctx context.Context, name string, sType ServiceType) error    { return nil }
func (m *mockManager) Restart(ctx context.Context, name string, sType ServiceType) error { return nil }
func (m *mockManager) Enable(ctx context.Context, name string, sType ServiceType) error  { return nil }
func (m *mockManager) Disable(ctx context.Context, name string, sType ServiceType) error { return nil }
func (m *mockManager) GenerateConfig(cfg ServiceConfig) (string, error)                  { return "mock-config", nil }
func (m *mockManager) GetConfigPath(name string, sType ServiceType) string               { return "/mock/path" }
func (m *mockManager) InstallService(ctx context.Context, cfg ServiceConfig, content string, enableNow bool) (string, error) {
	return "/mock/path", nil
}
func (m *mockManager) DeleteService(ctx context.Context, name string, sType ServiceType) error {
	return nil
}
func (m *mockManager) StreamLogs(ctx context.Context, name string, sType ServiceType, lines int, follow bool) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "mock log"
	close(ch)
	return ch, nil
}

func TestRegistryWithMock(t *testing.T) {
	mock := &mockManager{name: "test-init", available: true}
	Register("test-init", mock)

	m, err := Get("test-init")
	if err != nil {
		t.Fatalf("failed to retrieve registered manager: %v", err)
	}

	if m.Name() != "test-init" {
		t.Errorf("expected name 'test-init', got %q", m.Name())
	}

	svcs, err := m.ListServices(context.Background(), TypeUser)
	if err != nil || len(svcs) == 0 {
		t.Errorf("failed to list mock services: %v", err)
	}
}
