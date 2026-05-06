// Package systemctl provides shared helpers for the systemctl CLI shim.
package systemctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shim-systemctl/pkg/runit"
)

const MaskedDir = "/etc/sins/masked"

type Ctx struct {
	Mgr       *runit.Manager
	UnitPaths []string
	Quiet     bool
	UserMode  bool
}

func NewCtx(mgr *runit.Manager, userMode, quiet bool) *Ctx {
	return &Ctx{
		Mgr:       mgr,
		UnitPaths: ResolveUnitPaths(userMode),
		Quiet:     quiet,
		UserMode:  userMode,
	}
}

func ResolveUnitPaths(userMode bool) []string {
	if custom := os.Getenv("SYSTEMD_UNIT_PATH"); custom != "" {
		return strings.Split(custom, ":")
	}
	var paths []string
	if userMode {
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, ".config/systemd/user"),
				filepath.Join(home, ".local/share/systemd/user"),
			)
		}
	}
	paths = append(paths, "/etc/systemd/system", "/usr/lib/systemd/system")
	return paths
}

func (c *Ctx) FindUnitFile(name string) string {
	ext := false
	for _, suf := range []string{".service", ".timer", ".socket"} {
		if strings.HasSuffix(name, suf) {
			ext = true
			break
		}
	}
	if !ext {
		name += ".service"
	}
	for _, p := range c.UnitPaths {
		path := filepath.Join(p, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func normalizeMaskedBase(unit string) string {
	base := filepath.Base(unit)
	base = strings.TrimSuffix(base, ".service")
	base = strings.TrimSuffix(base, ".timer")
	base = strings.TrimSuffix(base, ".socket")
	return base + ".service"
}

func (c *Ctx) MaskedPath(unit string) string {
	return filepath.Join(MaskedDir, normalizeMaskedBase(unit))
}

func (c *Ctx) IsMasked(unit string) bool {
	_, err := os.Stat(c.MaskedPath(unit))
	return err == nil
}

func (c *Ctx) Mask(unit string) error {
	if err := os.MkdirAll(MaskedDir, 0755); err != nil {
		return err
	}
	p := c.MaskedPath(unit)
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	_, _ = f.WriteString("# masked by SINS systemctl\n")
	return f.Close()
}

func (c *Ctx) Unmask(unit string) error {
	err := os.Remove(c.MaskedPath(unit))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (c *Ctx) Logf(format string, args ...interface{}) {
	if c.Quiet {
		return
	}
	fmt.Printf(format, args...)
}
