package updater

import (
	"context"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
)

var (
	// Version is populated at link time via -ldflags
	Version = "v0.1.0-dev"
	// Commit is populated at link time
	Commit = "none"
	// BuildDate is populated at link time
	BuildDate = "unknown"

	// DefaultRepo is the GitHub repository for updates
	DefaultRepo = "khameleonium/alirun"
)

// Info returns a formatted string with version and build details
func Info() string {
	return fmt.Sprintf("Alirun %s (commit: %s, built: %s, %s/%s, %s)",
		Version, Commit, BuildDate, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// UpdateResult holds information about an update check or application
type UpdateResult struct {
	CurrentVersion string
	LatestVersion  string
	Found          bool
	ReleaseNotes   string
	URL            string
}

// CheckForUpdate checks if a newer version exists on GitHub
func CheckForUpdate(ctx context.Context, repo string) (*UpdateResult, error) {
	if repo == "" {
		repo = DefaultRepo
	}

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	result := &UpdateResult{
		CurrentVersion: Version,
		Found:          found,
	}

	if found && latest != nil {
		result.LatestVersion = latest.Version()
		result.ReleaseNotes = latest.ReleaseNotes
		result.URL = latest.URL
	}

	return result, nil
}

// SelfUpdate checks and updates the binary in-place
func SelfUpdate(ctx context.Context, repo string) (*UpdateResult, error) {
	if repo == "" {
		repo = DefaultRepo
	}

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	if !found || latest == nil {
		return &UpdateResult{
			CurrentVersion: Version,
			Found:          false,
		}, nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("could not determine executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	return &UpdateResult{
		CurrentVersion: Version,
		LatestVersion:  latest.Version(),
		Found:          true,
		ReleaseNotes:   latest.ReleaseNotes,
		URL:            latest.URL,
	}, nil
}
