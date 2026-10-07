package detector

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileType represents the detected type of the file
type FileType string

const (
	FileTypeBinary FileType = "Native Binary (ELF)"
	FileTypeBash   FileType = "Bash Script"
	FileTypePython FileType = "Python Script"
	FileTypeNode   FileType = "Node.js Script"
	FileTypeOther  FileType = "Executable Script"
)

// TargetInspection holds auto-detected properties of a target executable or script
type TargetInspection struct {
	OriginalPath     string
	AbsolutePath     string
	Exists           bool
	IsExecutable     bool
	SuggestedName    string
	SuggestedWorkDir string
	DetectedType     FileType
	Interpreter      string
	ExecStartCommand string
	VirtualEnvPath   string
	DetectedEnvVars  map[string]string
	Warnings         []string
}

// Inspect analyzes the given file path or command line and prepares service recommendations
func Inspect(rawPath string) (*TargetInspection, error) {
	cleanPath := strings.TrimSpace(rawPath)
	if cleanPath == "" {
		return nil, fmt.Errorf("empty path provided")
	}

	// Check if input is a command line with arguments (e.g. "python3 -m http.server 8080" or "./script.sh --flag")
	words := strings.Fields(cleanPath)
	firstWord := words[0]
	hasArgs := len(words) > 1

	// Expand ~ in first word or path
	if strings.HasPrefix(firstWord, "~/") || firstWord == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			firstWord = filepath.Join(home, firstWord[1:])
			if hasArgs {
				words[0] = firstWord
				cleanPath = strings.Join(words, " ")
			} else {
				cleanPath = firstWord
			}
		}
	}

	// Try to resolve first word as absolute or relative file, or in PATH
	targetFile := firstWord
	absPath, err := filepath.Abs(targetFile)
	if err == nil {
		if _, statErr := os.Stat(absPath); statErr == nil {
			targetFile = absPath
		} else if lookPath, lookErr := exec.LookPath(firstWord); lookErr == nil {
			targetFile = lookPath
			absPath = lookPath
		}
	}

	info, statErr := os.Stat(targetFile)
	if statErr != nil {
		// File does not exist as path, check PATH
		if lookPath, lookErr := exec.LookPath(firstWord); lookErr == nil {
			targetFile = lookPath
			absPath = lookPath
			info, statErr = os.Stat(targetFile)
		}
	}

	cwd, _ := os.Getwd()

	if statErr != nil {
		// Command not found on disk, but treat as custom command
		base := filepath.Base(firstWord)
		sugName := sanitizeServiceName(base)
		return &TargetInspection{
			OriginalPath:     rawPath,
			AbsolutePath:     targetFile,
			Exists:           false,
			SuggestedName:    sugName,
			SuggestedWorkDir: cwd,
			DetectedType:     FileTypeOther,
			ExecStartCommand: cleanPath,
			Warnings:         []string{fmt.Sprintf("Command not found in PATH: %s", firstWord)},
		}, nil
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not an executable file: %s", targetFile)
	}

	isExec := (info.Mode().Perm() & 0111) != 0
	base := filepath.Base(targetFile)
	sugName := sanitizeServiceName(base)
	workDir := filepath.Dir(targetFile)
	if workDir == "/usr/bin" || workDir == "/bin" || workDir == "/usr/local/bin" {
		workDir = cwd
	}

	// Build full ExecStart
	fullExecStart := cleanPath
	if hasArgs {
		words[0] = targetFile
		fullExecStart = strings.Join(words, " ")
	} else {
		fullExecStart = targetFile
	}

	inspection := &TargetInspection{
		OriginalPath:     rawPath,
		AbsolutePath:     targetFile,
		Exists:           true,
		IsExecutable:     isExec,
		SuggestedName:    sugName,
		SuggestedWorkDir: workDir,
		ExecStartCommand: fullExecStart,
		DetectedEnvVars:  make(map[string]string),
	}

	if !isExec {
		inspection.Warnings = append(inspection.Warnings, "File is not marked executable (chmod +x is recommended)")
	}

	// Read file header to detect ELF or Shebang
	file, err := os.Open(absPath)
	if err == nil {
		defer file.Close()

		var header [512]byte
		n, _ := file.Read(header[:])

		if n >= 4 && bytes.Equal(header[:4], []byte{0x7f, 'E', 'L', 'F'}) {
			inspection.DetectedType = FileTypeBinary
			if !hasArgs {
				inspection.ExecStartCommand = absPath
			}
		} else {
			// Read first line (shebang)
			file.Seek(0, 0)
			scanner := bufio.NewScanner(file)
			if scanner.Scan() {
				firstLine := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(firstLine, "#!") {
					shebang := strings.TrimPrefix(firstLine, "#!")
					shebang = strings.TrimSpace(shebang)
					inspection.Interpreter = shebang

					if strings.Contains(shebang, "python") {
						inspection.DetectedType = FileTypePython
					} else if strings.Contains(shebang, "bash") || strings.Contains(shebang, "sh") {
						inspection.DetectedType = FileTypeBash
					} else if strings.Contains(shebang, "node") {
						inspection.DetectedType = FileTypeNode
					} else {
						inspection.DetectedType = FileTypeOther
					}
				}
			}
		}
	}

	// Check if Python script or extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == ".py" {
		inspection.DetectedType = FileTypePython
	} else if ext == ".sh" {
		inspection.DetectedType = FileTypeBash
	} else if ext == ".js" || ext == ".mjs" {
		inspection.DetectedType = FileTypeNode
	}

	// Check for local virtual environments for Python scripts
	if inspection.DetectedType == FileTypePython && !hasArgs {
		venvPython := findVirtualEnvPython(workDir)
		if venvPython != "" {
			inspection.VirtualEnvPath = filepath.Dir(filepath.Dir(venvPython))
			inspection.ExecStartCommand = fmt.Sprintf("%s %s", venvPython, absPath)
		} else if inspection.IsExecutable && inspection.Interpreter != "" {
			inspection.ExecStartCommand = absPath
		} else {
			inspection.ExecStartCommand = fmt.Sprintf("/usr/bin/python3 %s", absPath)
		}
	} else if inspection.DetectedType == FileTypeBash && !hasArgs {
		if inspection.IsExecutable {
			inspection.ExecStartCommand = absPath
		} else {
			inspection.ExecStartCommand = fmt.Sprintf("/bin/bash %s", absPath)
		}
	} else if inspection.DetectedType == FileTypeNode && !hasArgs {
		if inspection.IsExecutable && inspection.Interpreter != "" {
			inspection.ExecStartCommand = absPath
		} else {
			nodePath, err := checkCommandPath("node")
			if err == nil {
				inspection.ExecStartCommand = fmt.Sprintf("%s %s", nodePath, absPath)
			} else {
				inspection.ExecStartCommand = fmt.Sprintf("node %s", absPath)
			}
		}
	} else if !hasArgs {
		inspection.ExecStartCommand = absPath
	}

	// Check for .env file in the directory
	dotEnvPath := filepath.Join(workDir, ".env")
	if _, err := os.Stat(dotEnvPath); err == nil {
		inspection.Warnings = append(inspection.Warnings, fmt.Sprintf("Found .env file in %s (can be loaded via EnvironmentFile)", workDir))
	}

	return inspection, nil
}

// MakeExecutable adds execute permission (+x) to the file
func MakeExecutable(absPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	// Like "chmod +x" with a typical umask: grant execute only to those who can already
	// read the file (0600 -> 0700, 0644 -> 0755), never widen read/write access.
	mode := info.Mode()
	newMode := mode | (mode&0444)>>2 | 0100
	return os.Chmod(absPath, newMode)
}

func sanitizeServiceName(name string) string {
	ext := filepath.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, ext)
	nameWithoutExt = strings.ToLower(nameWithoutExt)

	var sb strings.Builder
	lastHyphen := false
	for _, r := range nameWithoutExt {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			lastHyphen = false
		} else {
			if !lastHyphen {
				sb.WriteRune('-')
				lastHyphen = true
			}
		}
	}

	res := strings.Trim(sb.String(), "-")
	if res == "" {
		return "custom-service"
	}
	return res
}

func findVirtualEnvPython(dir string) string {
	candidates := []string{
		filepath.Join(dir, ".venv", "bin", "python"),
		filepath.Join(dir, "venv", "bin", "python"),
		filepath.Join(dir, "env", "bin", "python"),
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			if (info.Mode().Perm() & 0111) != 0 {
				return c
			}
		}
	}
	return ""
}

func checkCommandPath(cmd string) (string, error) {
	// Look up in PATH
	paths := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range paths {
		full := filepath.Join(p, cmd)
		if info, err := os.Stat(full); err == nil && !info.IsDir() && (info.Mode().Perm()&0111 != 0) {
			return full, nil
		}
	}
	return "", fmt.Errorf("command %s not found in PATH", cmd)
}
