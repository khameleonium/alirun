package updater

import (
	"context"
	"fmt"
	"runtime"

	"github.com/Masterminds/semver/v3"
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
	return fmt.Sprintf("Alirun %s (commit: %s, built: %s, %s/%s, %s)\nСоздатель: Илья Ульянов | khameleonium",
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

	newer, err := isNewerRelease(latest, found)
	if err != nil {
		return nil, err
	}

	result := &UpdateResult{
		CurrentVersion: Version,
		Found:          newer,
	}

	if newer {
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

	newer, err := isNewerRelease(latest, found)
	if err != nil {
		return nil, err
	}
	if !newer {
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

// isNewerRelease reports whether the detected release is strictly newer than the running binary.
// DetectLatest only tells that a release exists, so the version comparison has to be done here,
// otherwise the same (or an older) release would be re-installed on every run.
func isNewerRelease(latest *selfupdate.Release, found bool) (bool, error) {
	if !found || latest == nil {
		return false, nil
	}
	return isNewerVersion(latest.Version(), Version)
}

// isNewerVersion compares two semantic versions ("v1.2.3" or "1.2.3")
func isNewerVersion(latest, current string) (bool, error) {
	cur, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("current version %q is not a release version (development build); install a release build to use self-update", current)
	}
	lat, err := semver.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("invalid release version %q: %w", latest, err)
	}
	return lat.GreaterThan(cur), nil
}
