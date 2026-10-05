package openrc

import (
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"os"
	"os/exec"
)

func init() {
	initsys.Register("openrc", New())
}

// Manager manages OpenRC services (/etc/init.d/)
type Manager struct{}

// New creates a new OpenRC manager
func New() *Manager {
	return &Manager{}
}

func (m *Manager) Name() string {
	return "openrc"
}

func (m *Manager) IsAvailable() bool {
	if _, err := os.Stat("/run/openrc"); err == nil {
		return true
	}
	_, err := exec.LookPath("rc-service")
	return err == nil
}

func (m *Manager) GetConfigPath(name string, sType initsys.ServiceType) string {
	return "/etc/init.d/" + name
}

func (m *Manager) ListServices(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	return nil, fmt.Errorf("openrc module is currently in development (contributions welcome at github.com/khameleonium/alirun)")
}

func (m *Manager) GetStatus(ctx context.Context, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	return nil, fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) Start(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) Stop(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) Restart(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) Enable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) Disable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) GenerateConfig(cfg initsys.ServiceConfig) (string, error) {
	return fmt.Sprintf("#!/sbin/openrc-run\n# OpenRC service script for %s\ncommand=\"%s\"\n", cfg.Name, cfg.ExecStart), nil
}

func (m *Manager) InstallService(ctx context.Context, cfg initsys.ServiceConfig, content string, enableNow bool) (string, error) {
	return "", fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	return fmt.Errorf("openrc module is currently in development")
}

func (m *Manager) StreamLogs(ctx context.Context, name string, sType initsys.ServiceType, lines int, follow bool) (<-chan string, error) {
	return nil, fmt.Errorf("openrc module is currently in development")
}
