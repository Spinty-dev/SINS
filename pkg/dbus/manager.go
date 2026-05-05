//go:build dbus
package dbus

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"shim-systemctl/pkg/runit"
)

type SinsManager struct {
	runitMgr *runit.Manager
}

func Register(conn *dbus.Conn, rm *runit.Manager) error {
	m := &SinsManager{runitMgr: rm}
	
	// Register Systemd Manager
	if err := conn.Export(m, "/org/freedesktop/systemd1", "org.freedesktop.systemd1.Manager"); err != nil {
		return err
	}

	// Register Hostnamed
	if err := conn.Export(m, "/org/freedesktop/hostname1", "org.freedesktop.hostname1"); err != nil {
		return err
	}

	// Register Timedated
	if err := conn.Export(m, "/org/freedesktop/timedate1", "org.freedesktop.timedate1"); err != nil {
		return err
	}

	// Register Localed
	if err := conn.Export(m, "/org/freedesktop/locale1", "org.freedesktop.locale1"); err != nil {
		return err
	}

	node := introspect.Node{
		Name: "/org/freedesktop/systemd1",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name: "org.freedesktop.systemd1.Manager",
				Methods: []introspect.Method{
					{Name: "ListUnits", Args: []introspect.Arg{{Name: "units", Type: "a(ssssssouso)", Direction: "out"}}},
					{Name: "StartUnit", Args: []introspect.Arg{{Name: "name", Type: "s", Direction: "in"}, {Name: "mode", Type: "s", Direction: "in"}, {Name: "job", Type: "o", Direction: "out"}}},
					{Name: "StopUnit", Args: []introspect.Arg{{Name: "name", Type: "s", Direction: "in"}, {Name: "mode", Type: "s", Direction: "in"}, {Name: "job", Type: "o", Direction: "out"}}},
					{Name: "RestartUnit", Args: []introspect.Arg{{Name: "name", Type: "s", Direction: "in"}, {Name: "mode", Type: "s", Direction: "in"}, {Name: "job", Type: "o", Direction: "out"}}},
					{Name: "GetUnit", Args: []introspect.Arg{{Name: "name", Type: "s", Direction: "in"}, {Name: "unit", Type: "o", Direction: "out"}}},
				},
			},
		},
	}

	return conn.Export(introspect.NewIntrospectable(&node), "/org/freedesktop/systemd1", "org.freedesktop.DBus.Introspectable")
}

// Systemd Methods
type UnitInfo struct {
	Name        string
	Description string
	LoadState   string
	ActiveState string
	SubState    string
	Followed    string
	Path        dbus.ObjectPath
	JobId       uint32
	JobType     string
	JobPath     dbus.ObjectPath
}

func escapePath(name string) string {
	res := strings.ReplaceAll(name, "-", "_")
	res = strings.ReplaceAll(res, ".", "_2e")
	return res
}

func (m *SinsManager) ListUnits() ([]UnitInfo, *dbus.Error) {
	files, err := os.ReadDir(m.runitMgr.ServiceDir)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	var units []UnitInfo
	for _, f := range files {
		if !f.IsDir() { continue }
		name := f.Name()
		activeState := "inactive"
		statusFile := filepath.Join(m.runitMgr.ServiceDir, name, "supervise", "stat")
		if data, err := os.ReadFile(statusFile); err == nil {
			if strings.HasPrefix(string(data), "run") { activeState = "active" }
		}
		units = append(units, UnitInfo{
			Name: name + ".service", Description: "SINS Managed Service", LoadState: "loaded",
			ActiveState: activeState, SubState: activeState, Path: dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + escapePath(name)),
			JobPath: dbus.ObjectPath("/"),
		})
	}
	return units, nil
}

func (m *SinsManager) StartUnit(name string, mode string) (dbus.ObjectPath, *dbus.Error) {
	cleanName := strings.TrimSuffix(name, ".service")
	m.runitMgr.EnableService(cleanName)
	exec.Command("sv", "start", cleanName).Run()
	return "/", nil
}

func (m *SinsManager) StopUnit(name string, mode string) (dbus.ObjectPath, *dbus.Error) {
	cleanName := strings.TrimSuffix(name, ".service")
	exec.Command("sv", "stop", cleanName).Run()
	return "/", nil
}

func (m *SinsManager) RestartUnit(name string, mode string) (dbus.ObjectPath, *dbus.Error) {
	cleanName := strings.TrimSuffix(name, ".service")
	exec.Command("sv", "restart", cleanName).Run()
	return "/", nil
}

func (m *SinsManager) GetUnit(name string) (dbus.ObjectPath, *dbus.Error) {
	cleanName := strings.TrimSuffix(name, ".service")
	return dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + escapePath(cleanName)), nil
}

// Hostnamed Methods
func (m *SinsManager) SetHostname(name string, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetStaticHostname(name string, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetPrettyHostname(name string, interactive bool) *dbus.Error { return nil }

// Timedated Methods
func (m *SinsManager) SetTime(usec int64, relative, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetTimezone(timezone string, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetLocalRTC(local_rtc, fix_system, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetNTP(use_ntp, interactive bool) *dbus.Error { return nil }

// Localed Methods
func (m *SinsManager) SetLocale(locale []string, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetVConsoleKeyboard(keymap, keymap_toggle string, convert, interactive bool) *dbus.Error { return nil }
func (m *SinsManager) SetX11Keyboard(layouts, model, variant, options string, convert, interactive bool) *dbus.Error { return nil }
