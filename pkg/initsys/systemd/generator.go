package systemd

import (
	"alirun/pkg/initsys"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/coreos/go-systemd/v22/unit"
)

// GenerateUnitFile generates valid systemd .service unit content using coreos/go-systemd/v22/unit
func GenerateUnitFile(cfg initsys.ServiceConfig) (string, error) {
	if strings.TrimSpace(cfg.Name) == "" {
		return "", fmt.Errorf("service name cannot be empty")
	}
	if strings.TrimSpace(cfg.ExecStart) == "" {
		return "", fmt.Errorf("ExecStart command cannot be empty")
	}

	var opts []*unit.UnitOption

	// --- [Unit] Section ---
	desc := cfg.Description
	if desc == "" {
		desc = fmt.Sprintf("%s (managed by Alirun)", cfg.Name)
	}
	opts = append(opts, unit.NewUnitOption("Unit", "Description", desc))

	if cfg.WantsNetwork || cfg.Preset == initsys.PresetWeb {
		opts = append(opts, unit.NewUnitOption("Unit", "After", "network-online.target"))
		opts = append(opts, unit.NewUnitOption("Unit", "Wants", "network-online.target"))
	}

	// --- [Service] Section ---
	switch cfg.Preset {
	case initsys.PresetOneshot:
		opts = append(opts, unit.NewUnitOption("Service", "Type", "oneshot"))
		opts = append(opts, unit.NewUnitOption("Service", "RemainAfterExit", "yes"))
	default:
		opts = append(opts, unit.NewUnitOption("Service", "Type", "simple"))
	}

	opts = append(opts, unit.NewUnitOption("Service", "ExecStart", cfg.ExecStart))

	if cfg.WorkingDirectory != "" {
		opts = append(opts, unit.NewUnitOption("Service", "WorkingDirectory", cfg.WorkingDirectory))
	}

	// Restart policy
	restartPolicy := cfg.RestartPolicy
	if restartPolicy == "" {
		if cfg.Preset == initsys.PresetDaemon || cfg.Preset == initsys.PresetWeb {
			restartPolicy = "always"
		}
	}

	if restartPolicy != "" && cfg.Preset != initsys.PresetOneshot {
		opts = append(opts, unit.NewUnitOption("Service", "Restart", restartPolicy))
		sec := cfg.RestartSec
		if sec <= 0 {
			sec = 5
		}
		opts = append(opts, unit.NewUnitOption("Service", "RestartSec", fmt.Sprintf("%ds", sec)))
	}

	// Environment variables
	if len(cfg.Environment) > 0 {
		// Sort keys for deterministic output
		keys := make([]string, 0, len(cfg.Environment))
		for k := range cfg.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := cfg.Environment[k]
			opts = append(opts, unit.NewUnitOption("Service", "Environment", fmt.Sprintf("%s=%s", k, v)))
		}
	}

	if cfg.EnvironmentFile != "" {
		opts = append(opts, unit.NewUnitOption("Service", "EnvironmentFile", cfg.EnvironmentFile))
	}

	// Standard streams
	opts = append(opts, unit.NewUnitOption("Service", "StandardOutput", "journal"))
	opts = append(opts, unit.NewUnitOption("Service", "StandardError", "journal"))

	// --- [Install] Section ---
	wantedBy := "default.target"
	if cfg.Type == initsys.TypeSystem {
		wantedBy = "multi-user.target"
	}
	opts = append(opts, unit.NewUnitOption("Install", "WantedBy", wantedBy))

	reader := unit.Serialize(opts)
	buf, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to serialize unit: %w", err)
	}

	return string(buf), nil
}
