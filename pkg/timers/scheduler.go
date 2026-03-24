package timers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shim-systemctl/pkg/units"
	"strings"
	"time"
)

type Timer struct {
	Unit     *units.Unit
	Name     string
	Target   string
	NextRun  time.Time
	Interval time.Duration
}

func NewTimer(path string) (*Timer, error) {
	unit, err := units.Parse(path)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(path)
	target := unit.Get("Timer", "Unit")
	if target == "" {
		target = strings.TrimSuffix(name, ".timer") + ".service"
	}

	t := &Timer{
		Unit: unit,
		Name: name,
		Target: target,
	}
	t.UpdateNextRun()
	return t, nil
}

func (t *Timer) UpdateNextRun() {
	calendar := t.Unit.Get("Timer", "OnCalendar")
	if calendar != "" {
		t.Interval = parseCalendar(calendar)
	} else {
		activeSec := t.Unit.Get("Timer", "OnUnitActiveSec")
		if activeSec != "" {
			t.Interval, _ = time.ParseDuration(strings.ReplaceAll(activeSec, " ", ""))
		}
	}

	if t.Interval == 0 {
		t.Interval = 1 * time.Hour
	}
	t.NextRun = time.Now().Add(t.Interval)
	fmt.Printf("Timer %s scheduled for %v\n", t.Name, t.NextRun)
}

func parseCalendar(val string) time.Duration {
	switch val {
	case "minutely":
		return 1 * time.Minute
	case "hourly":
		return 1 * time.Hour
	case "daily":
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(val)
	if err == nil {
		return d
	}
	return 0
}

func (t *Timer) Trigger() {
	fmt.Printf("[%v] Triggering timer %s -> %s\n", time.Now().Format(time.RFC3339), t.Name, t.Target)
	
	systemctlPath := os.Getenv("SYSTEMCTL_PATH")
	if systemctlPath == "" {
		systemctlPath = "systemctl"
	}

	cmd := exec.Command(systemctlPath, "start", t.Target)
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error triggering %s: %v\n", t.Target, err)
	}
}
