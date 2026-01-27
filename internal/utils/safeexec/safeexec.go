package safeexec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LookPath searches for an executable in the directories named by the PATH environment variable.
// It acts as a drop-in replacement for exec.LookPath but avoids using faccessat2 on Linux,
// which causes SIGSYS crashes on some Android/Termux kernels due to seccomp filtering.
func LookPath(file string) (string, error) {
	// If it contains a separator, it's relative or absolute, use standard logic but check safely
	if strings.Contains(file, string(filepath.Separator)) {
		info, err := os.Stat(file)
		if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return file, nil
		}
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}

	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, file)
		info, err := os.Stat(path)
		// Check if file exists, is not a dir, and is executable (bit 0111)
		// os.Stat uses lighter syscalls (fstat) than exec.LookPath (faccessat2)
		if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return path, nil
		}
	}

	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

// Command returns the Cmd struct to execute the named program with the given arguments.
// It resolves the executable path using SafeLookPath to prevent SIGSYS crashes on restricted systems.
func Command(name string, arg ...string) *exec.Cmd {
	// Try to resolve the path safely
	path, err := LookPath(name)
	if err == nil {
		// If found, use the resolved absolute path
		return exec.Command(path, arg...)
	}
	
	// If not found, fallback to standard behavior (which might crash later or fail)
	return exec.Command(name, arg...)
}
