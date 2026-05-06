package systemctl

import (
	"fmt"
	"os"
	"strings"
)

// UserModeBanner is printed once when --user is used (stderr) so AUR/desktop users know limits.
func UserModeBanner() string {
	var b strings.Builder
	b.WriteString("SINS (--user): user unit files are read from ~/.config/systemd/user and ~/.local/share/systemd/user (and SYSTEMD_UNIT_PATH).\n")
	b.WriteString("Runit services managed by SINS are system-wide under RUNIT_SV_DIR (default /etc/runit/sv). ")
	b.WriteString("`systemctl --user start|enable|mask` cannot install a private runit tree — use system units or symlink services yourself.\n")
	b.WriteString("Commands that only read unit files (status, show, cat, list-unit-files) work with --user.\n")
	return b.String()
}

// UserModeBlocksMutation reports commands that would misleadingly touch system runit while --user was passed.
func UserModeBlocksMutation(cmd string) bool {
	switch cmd {
	case "start", "stop", "restart", "try-restart", "reload", "reload-or-restart",
		"try-reload-or-restart", "enable", "disable", "mask", "unmask",
		"kill", "preset":
		return true
	default:
		return false
	}
}

// PrintUserMutationError explains exit to stderr.
func PrintUserMutationError(cmd string) {
	fmt.Fprintf(os.Stderr, "SINS: command %q is not supported with --user (no per-user runit generation). Omit --user or use a system-wide service.\n", cmd)
}
