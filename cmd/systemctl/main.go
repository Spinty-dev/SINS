package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shim-systemctl/pkg/runit"
	"shim-systemctl/pkg/units"
	"strings"
)

var unitPaths = []string{
	"/etc/systemd/system",
	"/usr/lib/systemd/system",
}

func getUnitPaths() []string {
	custom := os.Getenv("SYSTEMD_UNIT_PATH")
	if custom != "" {
		return strings.Split(custom, ":")
	}
	return unitPaths
}

func findUnitFile(name string) string {
	if !strings.HasSuffix(name, ".service") && !strings.HasSuffix(name, ".timer") && !strings.HasSuffix(name, ".socket") {
		name += ".service"
	}
	paths := getUnitPaths()
	for _, p := range paths {
		path := filepath.Join(p, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := ""
	unitName := ""
	flags := make(map[string]bool)

	// Robust argument parsing
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			// It's a flag
			if arg == "-f" || arg == "--follow" {
				flags["follow"] = true
			} else if arg == "--now" {
				flags["now"] = true
			} else {
				// Ignore other common flags for compatibility
				// e.g., --user, --system, --no-block, --no-pager
				flags[arg] = true
			}
		} else {
			// It's a positional argument
			if command == "" {
				command = arg
			} else if unitName == "" {
				unitName = arg
			}
		}
	}

	if command == "" {
		printUsage()
		os.Exit(1)
	}
	
	if command == "daemon-reload" {
		fmt.Println("Daemon reload (noop in shim)")
		return
	}

	if unitName == "" && command != "daemon-reload" {
		fmt.Printf("Unit name required for command %s\n", command)
		os.Exit(1)
	}

	mgr := runit.NewManager()

	switch command {
	case "start":
		handleAction(mgr, unitName, "start", true)
	case "stop":
		handleAction(mgr, unitName, "stop", false)
	case "restart":
		handleAction(mgr, unitName, "restart", true)
	case "reload":
		handleReload(mgr, unitName)
	case "status":
		handleStatus(mgr, unitName, flags["follow"])
	case "is-active":
		handleIsActive(mgr, unitName)
	case "is-enabled":
		handleIsEnabled(mgr, unitName)
	case "enable":
		handleEnable(mgr, unitName)
		if flags["now"] {
			handleAction(mgr, unitName, "start", true)
		}
	case "disable":
		if flags["now"] {
			handleAction(mgr, unitName, "stop", false)
		}
		handleDisable(mgr, unitName)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: systemctl <command> [unit]")
	fmt.Println("\nCommands:")
	fmt.Println("  start, stop, restart, reload, status, enable, disable")
	fmt.Println("  is-active, is-enabled, daemon-reload")
}

func ensureServiceExists(mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	svPath := filepath.Join(mgr.ServiceDir, cleanName)
	
	if _, err := os.Stat(svPath); os.IsNotExist(err) {
		err := units.DetectCycles(name, func(n string) (*units.Unit, error) {
			path := findUnitFile(n)
			if path == "" {
				return nil, nil
			}
			return units.Parse(path)
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		var unit *units.Unit

		// Handle templates (service@instance)
		if strings.Contains(cleanName, "@") {
			parts := strings.SplitN(cleanName, "@", 2)
			templateName := parts[0] + "@.service"
			instance := parts[1]
			
			unitPath := findUnitFile(templateName)
			if unitPath != "" {
				unit, err = units.Parse(unitPath)
				if err == nil {
					unit.ReplacePlaceholders(instance)
				}
			}
		}

		if unit == nil {
			unitPath := findUnitFile(name)
			if unitPath == "" {
				fmt.Printf("Unit %s not found\n", name)
				os.Exit(1)
			}
			unit, err = units.Parse(unitPath)
		}

		if err != nil {
			fmt.Printf("Failed to handle unit: %v\n", err)
			os.Exit(1)
		}
		
		if err := mgr.SetupService(cleanName, unit); err != nil {
			fmt.Printf("Failed to setup runit service: %v\n", err)
			os.Exit(1)
		}
	}
}

func handleAction(mgr *runit.Manager, name string, svCmd string, autoCreate bool) {
	cleanName := strings.TrimSuffix(name, ".service")
	
	if autoCreate {
		ensureServiceExists(mgr, name)
	}

	if err := mgr.WaitForService(cleanName, 5); err != nil {
		fmt.Printf("Warning: %v. Runit might still be initializing the service.\n", err)
	}

	cmd := exec.Command("sv", svCmd, cleanName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to execute sv %s: %v\n", svCmd, err)
		os.Exit(1)
	}
}

func handleReload(mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	
	cmd := exec.Command("sv", "status", cleanName)
	output, _ := cmd.CombinedOutput()
	if !strings.HasPrefix(string(output), "run:") {
		fmt.Printf("Service %s is not running, cannot reload.\n", name)
		os.Exit(3)
	}

	unitPath := findUnitFile(name)
	if unitPath != "" {
		unit, err := units.Parse(unitPath)
		if err == nil {
			execReload := unit.GetServiceField("ExecReload")
			if execReload != "" {
				fmt.Printf("Executing ExecReload: %s\n", execReload)
				parts := strings.Split(execReload, " ")
				reloadCmd := exec.Command(parts[0], parts[1:]...)
				reloadCmd.Stdout = os.Stdout
				reloadCmd.Stderr = os.Stderr
				if err := reloadCmd.Run(); err != nil {
					fmt.Printf("ExecReload failed: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}
	}

	fmt.Printf("No ExecReload found, sending SIGHUP via sv hup %s\n", cleanName)
	if err := exec.Command("sv", "hup", cleanName).Run(); err != nil {
		fmt.Printf("Failed to send SIGHUP: %v\n", err)
		os.Exit(1)
	}
}

func handleStatus(mgr *runit.Manager, name string, follow bool) {
	cleanName := strings.TrimSuffix(name, ".service")
	svPath := filepath.Join(mgr.ServiceDir, cleanName)
	
	if _, err := os.Stat(svPath); os.IsNotExist(err) {
		fmt.Printf("Unit %s could not be found.\n", name)
		os.Exit(4)
	}

	cmd := exec.Command("sv", "status", cleanName)
	output, _ := cmd.CombinedOutput()
	fmt.Print(string(output))
	isActive := strings.HasPrefix(string(output), "run:")

	logFile := filepath.Join(mgr.ServiceDir, cleanName, "log", "main", "current")
	if _, err := os.Stat(logFile); err == nil {
		fmt.Println("\nRecent logs:")
		args := []string{"-n", "10"}
		if follow {
			args = []string{"-f"}
		}
		args = append(args, logFile)
		
		cmd := exec.Command("tail", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	if isActive {
		os.Exit(0)
	}
	os.Exit(3)
}

func handleIsActive(mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	cmd := exec.Command("sv", "status", cleanName)
	output, _ := cmd.CombinedOutput()
	if strings.HasPrefix(string(output), "run:") {
		fmt.Println("active")
		os.Exit(0)
	}
	fmt.Println("inactive")
	os.Exit(3)
}

func handleIsEnabled(mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	enablePath := filepath.Join(mgr.EnableDir, cleanName)
	if _, err := os.Lstat(enablePath); err == nil {
		fmt.Println("enabled")
		os.Exit(0)
	}
	fmt.Println("disabled")
	os.Exit(1)
}

func handleEnable(mgr *runit.Manager, name string) {
	ensureServiceExists(mgr, name)
	cleanName := strings.TrimSuffix(name, ".service")
	if err := mgr.EnableService(cleanName); err != nil {
		fmt.Printf("Failed to enable service: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created symlink for %s\n", cleanName)
}

func handleDisable(mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	if err := mgr.DisableService(cleanName); err != nil {
		fmt.Printf("Failed to disable service: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Removed symlink for %s\n", cleanName)
}
