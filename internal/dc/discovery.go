package dc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DiscoverRPCServer locates the deltachat-rpc-server binary using Parla's
// search order: explicit override, env var, PATH, local install folders, ~/.local/bin, ~/.cargo/bin.
func DiscoverRPCServer(explicitPath string) string {
	if explicitPath != "" && isExecutable(explicitPath) {
		return explicitPath
	}
	if envPath := os.Getenv("PARLA_RPC_SERVER"); envPath != "" && isExecutable(envPath) {
		return envPath
	}

	name := "deltachat-rpc-server"
	if runtime.GOOS == "windows" {
		name = "deltachat-rpc-server.exe"
	}

	// Check next to our own executable first (bundled Windows/macOS zip).
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), name)
		if isExecutable(sibling) {
			return sibling
		}
	}

	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	var candidates []string
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates,
				filepath.Join(local, "ParlaViva", name),
				filepath.Join(local, "Programs", "Parla", name),
				filepath.Join(local, "Parla", name),
			)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, ".cargo", "bin", name),
		)
	}
	for _, c := range candidates {
		if isExecutable(c) {
			return c
		}
	}
	return ""
}

// DefaultAccountsDir resolves the Delta Chat accounts directory, honoring
// DC_ACCOUNTS_PATH and platform-native Parla locations (%LOCALAPPDATA%\parla\accounts on Windows).
func DefaultAccountsDir() string {
	if p := os.Getenv("DC_ACCOUNTS_PATH"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "parla", "accounts")
		}
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "parla", "accounts")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "accounts"
	}
	return filepath.Join(home, ".local", "share", "parla", "accounts")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
