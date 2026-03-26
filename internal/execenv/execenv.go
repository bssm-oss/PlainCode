package execenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var commonToolDirs = []string{
	"/usr/local/go/bin",
	"/usr/local/bin",
	"/opt/homebrew/bin",
	"/Applications/Docker.app/Contents/Resources/bin",
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
}

// ResolveBinary returns an absolute binary path when a bare command name
// can be found in PATH or in common developer tool directories.
func ResolveBinary(name string) string {
	return resolveBinary(name, commonToolDirs)
}

// EnsurePath appends common developer tool directories to a PATH value.
func EnsurePath(current string) string {
	return ensurePath(current, commonToolDirs)
}

func resolveBinary(name string, searchDirs []string) string {
	if strings.TrimSpace(name) == "" {
		return name
	}
	if strings.ContainsRune(name, os.PathSeparator) {
		return name
	}
	if resolved, err := exec.LookPath(name); err == nil {
		return resolved
	}
	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		return candidate
	}
	return name
}

func ensurePath(current string, searchDirs []string) string {
	separator := string(os.PathListSeparator)
	seen := make(map[string]struct{})
	var parts []string

	for _, part := range strings.Split(current, separator) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		parts = append(parts, part)
	}

	for _, dir := range searchDirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		parts = append(parts, dir)
	}

	return strings.Join(parts, separator)
}
