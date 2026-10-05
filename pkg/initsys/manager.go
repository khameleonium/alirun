package initsys

import (
	"context"
	"time"
)

// ServiceType denotes user-level vs system-level service
type ServiceType string

const (
	TypeUser   ServiceType = "user"
	TypeSystem ServiceType = "system"
)

// ServiceStatus represents general service runtime status
type ServiceStatus string

const (
	StatusActive   ServiceStatus = "active"
	StatusInactive ServiceStatus = "inactive"
	StatusFailed   ServiceStatus = "failed"
	StatusUnknown  ServiceStatus = "unknown"
)

// PresetType defines common execution patterns
type PresetType string

const (
	PresetDaemon  PresetType = "daemon"  // Persistent background service (Restart=always)
	PresetOneshot PresetType = "oneshot" // Executes once and terminates (Type=oneshot)
	PresetWeb     PresetType = "web"     // Web service / server (After=network-online.target)
	PresetTimer   PresetType = "timer"   // Scheduled timer execution (service + timer)
)

// ServiceInfo holds normalized status information about any service
type ServiceInfo struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        ServiceType   `json:"type"`
	Status      ServiceStatus `json:"status"`
	SubState    string        `json:"sub_state"`
	Enabled     bool          `json:"enabled"`
	PID          int           `json:"pid"`
	MemoryBytes  uint64        `json:"memory_bytes"`
	CPUUsageNSec uint64        `json:"cpu_usage_nsec"`
	TasksCurrent uint64        `json:"tasks_current"`
	IsTimer      bool          `json:"is_timer"`
	TimerNext    string        `json:"timer_next"`
	ConfigPath   string        `json:"config_path"`
	ExecPath     string        `json:"exec_path"`
	ActiveSince  time.Time     `json:"active_since"`
	InitSystem   string        `json:"init_system"`
}

// ServiceConfig contains user inputs to generate a new service configuration
type ServiceConfig struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	ExecStart        string            `json:"exec_start"`
	WorkingDirectory string            `json:"working_directory"`
	Environment      map[string]string `json:"environment"`
	EnvironmentFile  string            `json:"environment_file"`
	RestartPolicy    string            `json:"restart_policy"`
	RestartSec       int               `json:"restart_sec"`
	Preset           PresetType        `json:"preset"`
	Type             ServiceType       `json:"type"`
	WantsNetwork     bool              `json:"wants_network"`
	TimerSchedule    string            `json:"timer_schedule"`
	TimerPersistent  bool              `json:"timer_persistent"`
}

// Manager is the abstract interface implemented by all init system providers
type Manager interface {
	// Name returns the identifier of the init system (e.g. "systemd", "openrc", "runit")
	Name() string

	// IsAvailable checks if this init system is active and accessible on the current host
	IsAvailable() bool

	// ListServices lists services managed by this init system
	ListServices(ctx context.Context, sType ServiceType) ([]ServiceInfo, error)

	// GetStatus retrieves detailed status for a specific service
	GetStatus(ctx context.Context, name string, sType ServiceType) (*ServiceInfo, error)

	// Start starts the service
	Start(ctx context.Context, name string, sType ServiceType) error

	// Stop stops the service
	Stop(ctx context.Context, name string, sType ServiceType) error

	// Restart restarts the service
	Restart(ctx context.Context, name string, sType ServiceType) error

	// Enable enables the service to start automatically
	Enable(ctx context.Context, name string, sType ServiceType) error

	// Disable disables autostart for the service
	Disable(ctx context.Context, name string, sType ServiceType) error

	// GenerateConfig builds the service unit configuration text
	GenerateConfig(cfg ServiceConfig) (string, error)

	// GetConfigPath returns where the configuration file should reside
	GetConfigPath(name string, sType ServiceType) string

	// InstallService writes configuration to disk, reloads daemon, and optionally enables it
	InstallService(ctx context.Context, cfg ServiceConfig, content string, enableNow bool) (string, error)

	// DeleteService stops, disables, deletes the config file, and reloads the daemon
	DeleteService(ctx context.Context, name string, sType ServiceType) error

	// StreamLogs streams log entries from the service via a Go channel
	StreamLogs(ctx context.Context, name string, sType ServiceType, lines int, follow bool) (<-chan string, error)
}
