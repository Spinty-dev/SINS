//go:build dbus
package dbus

import (
	"os"
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
	
	if err := conn.Export(m, "/org/freedesktop/systemd1", "org.freedesktop.systemd1.Manager"); err != nil {
		return err
	}

	node := introspect.Node{
		Name: "/org/freedesktop/systemd1",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name: "org.freedesktop.systemd1.Manager",
				Methods: []introspect.Method{
					{
						Name: "ListUnits",
						Args: []introspect.Arg{
							{Name: "units", Type: "a(ssssssouso)", Direction: "out"},
						},
					},
				},
			},
		},
	}

	return conn.Export(introspect.NewIntrospectable(&node), "/org/freedesktop/systemd1", "org.freedesktop.DBus.Introspectable")
}

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

func (m *SinsManager) ListUnits() ([]UnitInfo, *dbus.Error) {
	files, err := os.ReadDir(m.runitMgr.ServiceDir)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	var units []UnitInfo
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		
		name := f.Name()
		activeState := "inactive"
		
		statusFile := filepath.Join(m.runitMgr.ServiceDir, name, "supervise", "stat")
		if data, err := os.ReadFile(statusFile); err == nil {
			if strings.HasPrefix(string(data), "run") {
				activeState = "active"
			}
		}

		units = append(units, UnitInfo{
			Name:        name + ".service",
			Description: "SINS Managed Service",
			LoadState:   "loaded",
			ActiveState: activeState,
			SubState:    activeState,
			Path:        dbus.ObjectPath("/org/freedesktop/systemd1/unit/" + name),
		})
	}

	return units, nil
}
