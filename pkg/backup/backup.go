package backup

import (
	"alirun/pkg/initsys"
	"alirun/pkg/initsys/cron"
	"alirun/pkg/initsys/systemd"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BackupManifest contains serialized portable autostart and service configurations
type BackupManifest struct {
	Version   string       `yaml:"version" json:"version"`
	CreatedAt time.Time    `yaml:"created_at" json:"created_at"`
	Host      string       `yaml:"host" json:"host"`
	CreatedBy string       `yaml:"created_by" json:"created_by"`
	Author    string       `yaml:"author" json:"author"`
	Systemd   []UnitBackup `yaml:"systemd,omitempty" json:"systemd,omitempty"`
	Crontab   *CronBackup  `yaml:"crontab,omitempty" json:"crontab,omitempty"`
	XDG       []XDGBackup  `yaml:"xdg,omitempty" json:"xdg,omitempty"`
}

// UnitBackup stores a single systemd unit file and its metadata
type UnitBackup struct {
	Name    string `yaml:"name" json:"name"`
	Scope   string `yaml:"scope" json:"scope"` // "user" or "system"
	Path    string `yaml:"path" json:"path"`
	Content string `yaml:"content" json:"content"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// CronBackup stores user crontab and system cron files
type CronBackup struct {
	UserCrontab string            `yaml:"user_crontab,omitempty" json:"user_crontab,omitempty"`
	SystemFiles map[string]string `yaml:"system_files,omitempty" json:"system_files,omitempty"`
}

// XDGBackup stores a desktop autostart entry
type XDGBackup struct {
	Name    string `yaml:"name" json:"name"`
	Scope   string `yaml:"scope" json:"scope"` // "user" or "system"
	Path    string `yaml:"path" json:"path"`
	Content string `yaml:"content" json:"content"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// ImportResult contains the outcome of an import operation
type ImportResult struct {
	SystemdRestored []string `json:"systemd_restored"`
	CrontabRestored bool     `json:"crontab_restored"`
	XDGRestored     []string `json:"xdg_restored"`
	Errors          []string `json:"errors"`
}

// Export builds a BackupManifest for the given scope
func Export(ctx context.Context, sType initsys.ServiceType) (*BackupManifest, error) {
	hostname, _ := os.Hostname()
	manifest := &BackupManifest{
		Version:   "1.0",
		CreatedAt: time.Now().UTC(),
		Host:      hostname,
		CreatedBy: "Alirun Backup & Export Engine",
		Author:    "Создатель: Илья Ульянов | khameleonium",
	}

	// 1. Export Systemd Units
	var unitDirs []string
	if sType == initsys.TypeUser {
		home, _ := os.UserHomeDir()
		if home != "" {
			unitDirs = append(unitDirs, filepath.Join(home, ".config", "systemd", "user"))
		}
	} else {
		unitDirs = append(unitDirs, "/etc/systemd/system")
	}

	for _, dir := range unitDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".service") || strings.HasSuffix(name, ".timer") {
				fullPath := filepath.Join(dir, name)
				if e.Type()&os.ModeSymlink != 0 && !isPortableUnitLink(fullPath) {
					// Aliases, masks (/dev/null) and links to vendor units are managed by
					// systemd itself; restoring them as regular files would break them.
					continue
				}
				content, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				// Check if enabled
				enabled := false
				if sMgr, err := initsys.Get("systemd"); err == nil {
					if info, sErr := sMgr.GetStatus(ctx, name, sType); sErr == nil {
						enabled = info.Enabled
					}
				}

				manifest.Systemd = append(manifest.Systemd, UnitBackup{
					Name:    name,
					Scope:   string(sType),
					Path:    fullPath,
					Content: string(content),
					Enabled: enabled,
				})
			}
		}
	}

	// 2. Export Crontab
	if sType == initsys.TypeUser {
		crontabContent, _ := cron.ReadUserCrontab()
		if strings.TrimSpace(crontabContent) != "" {
			manifest.Crontab = &CronBackup{
				UserCrontab: crontabContent,
			}
		}
	} else {
		systemCron := make(map[string]string)
		if data, err := os.ReadFile("/etc/crontab"); err == nil {
			systemCron["/etc/crontab"] = string(data)
		}
		dFiles, _ := filepath.Glob("/etc/cron.d/*")
		for _, f := range dFiles {
			if data, err := os.ReadFile(f); err == nil {
				systemCron[f] = string(data)
			}
		}
		if len(systemCron) > 0 {
			manifest.Crontab = &CronBackup{
				SystemFiles: systemCron,
			}
		}
	}

	// 3. Export XDG Desktop Autostart
	var xdgDirs []string
	if sType == initsys.TypeUser {
		home, _ := os.UserHomeDir()
		if home != "" {
			xdgDirs = append(xdgDirs, filepath.Join(home, ".config", "autostart"))
		}
	} else {
		xdgDirs = append(xdgDirs, "/etc/xdg/autostart")
	}

	for _, dir := range xdgDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			fullPath := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			enabled := true
			if strings.Contains(string(data), "Hidden=true") || strings.Contains(string(data), "X-GNOME-Autostart-enabled=false") {
				enabled = false
			}

			manifest.XDG = append(manifest.XDG, XDGBackup{
				Name:    e.Name(),
				Scope:   string(sType),
				Path:    fullPath,
				Content: string(data),
				Enabled: enabled,
			})
		}
	}

	return manifest, nil
}

// isPortableUnitLink reports whether a symlinked unit points to a user-provided unit file
// (e.g. created by "systemctl link /path/app.service") whose content is worth exporting.
// Masked units, aliases and links into system/vendor unit directories are not portable.
func isPortableUnitLink(path string) bool {
	target, err := os.Readlink(path)
	if err != nil || !filepath.IsAbs(target) {
		return false // relative links are aliases within the unit directory
	}
	for _, prefix := range []string{"/dev/", "/usr/lib/", "/lib/", "/etc/systemd/", "/run/", "/usr/share/"} {
		if strings.HasPrefix(target, prefix) {
			return false
		}
	}
	return true
}

// safeBaseName validates that a name from a backup manifest is a plain file name
func safeBaseName(name string) (string, error) {
	base := filepath.Base(name)
	if name == "" || base != name || base == "." || base == ".." {
		return "", fmt.Errorf("invalid file name %q in backup", name)
	}
	return base, nil
}

// ToYAML serializes the manifest to formatted YAML
func (m *BackupManifest) ToYAML() ([]byte, error) {
	return yaml.Marshal(m)
}

// FromYAML parses a YAML byte slice into a BackupManifest
func FromYAML(data []byte) (*BackupManifest, error) {
	var m BackupManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid YAML backup manifest: %w", err)
	}
	return &m, nil
}

// Import restores configurations from a BackupManifest
func Import(ctx context.Context, manifest *BackupManifest, dryRun bool) (*ImportResult, error) {
	res := &ImportResult{}

	// 1. Restore Systemd Units
	home, _ := os.UserHomeDir()

	for _, u := range manifest.Systemd {
		// The destination is always derived from the scope and the current $HOME instead of
		// the absolute path stored in the backup, so backups can be restored on other machines
		// and for other user names.
		name, err := safeBaseName(u.Name)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		var destPath string
		if u.Scope == string(initsys.TypeUser) {
			if home == "" {
				res.Errors = append(res.Errors, fmt.Sprintf("cannot restore %s: home directory is unknown", name))
				continue
			}
			destPath = filepath.Join(home, ".config", "systemd", "user", name)
		} else {
			destPath = filepath.Join("/etc/systemd/system", name)
		}

		if dryRun {
			res.SystemdRestored = append(res.SystemdRestored, fmt.Sprintf("[dry-run] %s -> %s", u.Name, destPath))
			continue
		}

		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("failed to create dir %s: %v", dir, err))
			continue
		}

		if err := os.WriteFile(destPath, []byte(u.Content), 0644); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("failed to write unit %s: %v", destPath, err))
			continue
		}

		res.SystemdRestored = append(res.SystemdRestored, u.Name)

		// Optionally enable
		if u.Enabled {
			if sMgr, err := initsys.Get("systemd"); err == nil {
				_ = sMgr.Enable(ctx, u.Name, initsys.ServiceType(u.Scope))
			}
		}
	}

	// Reload systemd daemon if we modified systemd units
	if len(res.SystemdRestored) > 0 && !dryRun {
		if sMgr, err := initsys.Get("systemd"); err == nil {
			if s, ok := sMgr.(*systemd.Manager); ok {
				reloadCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()

				hasUser := false
				hasSystem := false
				for _, u := range manifest.Systemd {
					if u.Scope == string(initsys.TypeSystem) {
						hasSystem = true
					} else {
						hasUser = true
					}
				}
				if hasUser {
					_ = s.DaemonReload(reloadCtx, initsys.TypeUser)
				}
				if hasSystem && os.Geteuid() == 0 {
					_ = s.DaemonReload(reloadCtx, initsys.TypeSystem)
				}
			}
		}
	}

	// 2. Restore Crontab
	if manifest.Crontab != nil {
		if manifest.Crontab.UserCrontab != "" {
			if dryRun {
				res.CrontabRestored = true
			} else {
				if err := cron.SaveUserCrontab(manifest.Crontab.UserCrontab); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("failed to restore user crontab: %v", err))
				} else {
					res.CrontabRestored = true
				}
			}
		}

		if len(manifest.Crontab.SystemFiles) > 0 && !dryRun {
			for fPath, content := range manifest.Crontab.SystemFiles {
				clean := filepath.Clean(fPath)
				if clean != "/etc/crontab" && filepath.Dir(clean) != "/etc/cron.d" {
					res.Errors = append(res.Errors, fmt.Sprintf("refusing to restore unexpected cron path %s", fPath))
					continue
				}
				if err := os.WriteFile(clean, []byte(content), 0644); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("failed to restore %s: %v", fPath, err))
				}
			}
		}
	}

	// 3. Restore XDG Autostart
	for _, x := range manifest.XDG {
		name, err := safeBaseName(x.Name)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		var destPath string
		if x.Scope == string(initsys.TypeUser) {
			if home == "" {
				res.Errors = append(res.Errors, fmt.Sprintf("cannot restore %s: home directory is unknown", name))
				continue
			}
			destPath = filepath.Join(home, ".config", "autostart", name)
		} else {
			destPath = filepath.Join("/etc/xdg/autostart", name)
		}

		if dryRun {
			res.XDGRestored = append(res.XDGRestored, fmt.Sprintf("[dry-run] %s -> %s", x.Name, destPath))
			continue
		}

		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("failed to create dir %s: %v", dir, err))
			continue
		}

		if err := os.WriteFile(destPath, []byte(x.Content), 0644); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("failed to write desktop file %s: %v", destPath, err))
			continue
		}

		res.XDGRestored = append(res.XDGRestored, x.Name)
	}

	return res, nil
}
