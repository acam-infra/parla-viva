package dc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DiscoverRPCServer locates the deltachat-rpc-server binary using the same
// search order as Parla: explicit override, env var, PATH, ~/.local/bin, ~/.cargo/bin.
func DiscoverRPCServer(explicitPath string) string {
	if explicitPath != "" {
		if isExecutable(explicitPath) {
			return explicitPath
		}
	}
	if envPath := os.Getenv("PARLA_RPC_SERVER"); envPath != "" && isExecutable(envPath) {
		return envPath
	}

	name := "deltachat-rpc-server"
	if runtime.GOOS == "windows" {
		name = "deltachat-rpc-server.exe"
	}

	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, ".cargo", "bin", name),
	}
	for _, c := range candidates {
		if isExecutable(c) {
			return c
		}
	}
	return ""
}

// DefaultAccountsDir resolves the Delta Chat accounts directory, honoring
// DC_ACCOUNTS_PATH (matching Parla's shared account path policy).
func DefaultAccountsDir() string {
	if p := os.Getenv("DC_ACCOUNTS_PATH"); p != "" {
		return p
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
