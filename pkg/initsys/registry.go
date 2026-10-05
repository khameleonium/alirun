package initsys

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]Manager)
)

// Register registers an init system manager provider
func Register(name string, m Manager) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = m
}

// Get returns the registered manager by name
func Get(name string) (Manager, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()

	m, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("init system %q is not registered", name)
	}
	return m, nil
}

// ListRegistered returns all registered init system names
func ListRegistered() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()

	list := make([]string, 0, len(providers))
	for name := range providers {
		list = append(list, name)
	}
	return list
}

// Detect determines which init system is active on the host machine
func Detect() (Manager, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()

	// 1. Check if forced via environment variable
	if forced := os.Getenv("AUTOLIRUN_INIT"); forced != "" {
		if m, ok := providers[forced]; ok && m.IsAvailable() {
			return m, nil
		}
	}

	// 2. Check systemd presence (/run/systemd/system exists)
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		if m, ok := providers["systemd"]; ok && m.IsAvailable() {
			return m, nil
		}
	}

	// 3. Check OpenRC presence (/run/openrc exists)
	if _, err := os.Stat("/run/openrc"); err == nil {
		if m, ok := providers["openrc"]; ok && m.IsAvailable() {
			return m, nil
		}
	}

	// 4. Check Runit presence (/etc/runit or /run/runit.stopit)
	if _, err := os.Stat("/etc/runit"); err == nil {
		if m, ok := providers["runit"]; ok && m.IsAvailable() {
			return m, nil
		}
	}

	// 5. Fallback: check if systemctl command exists
	if _, err := exec.LookPath("systemctl"); err == nil {
		if m, ok := providers["systemd"]; ok && m.IsAvailable() {
			return m, nil
		}
	}

	return nil, fmt.Errorf("no supported init system detected on this system")
}
