package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shim-systemctl/pkg/runit"
	"shim-systemctl/pkg/safeunit"
	"shim-systemctl/pkg/systemctl"
	"shim-systemctl/pkg/units"
	"strings"
)

func asErrExit(err error, xe *systemctl.ErrExit) bool {
	return errors.As(err, xe)
}

// validateRunitServiceRef ensures argv cannot inject extra sv arguments or traverse paths.
func validateRunitServiceRef(unitName string) error {
	base := strings.TrimSuffix(unitName, ".service")
	base = strings.TrimSuffix(base, ".socket")
	base = strings.TrimSuffix(base, ".timer")
	return safeunit.ValidateServiceName(base)
}

func parseArgs() (cmd string, unitNames []string, userMode, quiet bool, now bool, follow bool) {
	pos := []string{}
	quiet = false
	userMode = false
	systemSeen := false
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-f", "--follow":
				follow = true
			case "--now":
				now = true
			case "--user":
				userMode = true
			case "--system":
				systemSeen = true
			case "-q", "--quiet":
				quiet = true
			}
			continue
		}
		pos = append(pos, arg)
	}
	if systemSeen {
		userMode = false
	}
	if len(pos) == 0 {
		return "", nil, userMode, quiet, now, follow
	}
	cmd = pos[0]
	if len(pos) > 1 {
		unitNames = append(unitNames, pos[1:]...)
	}
	return cmd, unitNames, userMode, quiet, now, follow
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command, unitNames, userMode, quiet, flagNow, follow := parseArgs()
	if command == "" {
		printUsage()
		os.Exit(1)
	}

	mgr := runit.NewManager()
	ctx := systemctl.NewCtx(mgr, userMode, quiet)

	if userMode && !quiet {
		fmt.Fprintln(os.Stderr, systemctl.UserModeBanner())
	}

	noUnitCommands := map[string]bool{
		"daemon-reload":     true,
		"list-units":        true,
		"list-unit-files":   true,
		"is-system-running": true,
		"preset-all":        true,
	}

	if noUnitCommands[command] {
		runGlobal(command, ctx, mgr)
		return
	}

	if len(unitNames) == 0 && command != "preset" {
		fmt.Fprintf(os.Stderr, "Unit name required for command %s\n", command)
		os.Exit(1)
	}

	if command == "preset" && len(unitNames) == 0 {
		if !quiet {
			fmt.Println("preset: no-op under SINS (no units given)")
		}
		os.Exit(0)
	}

	exit := 0
	for _, u := range unitNames {
		e := runUnitCommand(ctx, mgr, command, u, flagNow, follow)
		if e > exit {
			exit = e
		}
	}
	os.Exit(exit)
}

func runGlobal(command string, ctx *systemctl.Ctx, mgr *runit.Manager) {
	switch command {
	case "daemon-reload":
		ctx.Logf("Reloading units and regenerating run scripts...\n")
		files, err := os.ReadDir(mgr.ServiceDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading service directory: %v\n", err)
			os.Exit(1)
		}
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			name := f.Name()
			if err := safeunit.ValidateServiceName(name); err != nil {
				continue
			}
			unitPath := ctx.FindUnitFile(name)
			if unitPath != "" {
				unit, err := units.Parse(unitPath)
				if err == nil {
					ctx.Logf("Regenerating script for %s\n", name)
					_ = mgr.SetupService(name, unit)
				}
			}
		}
	case "list-units":
		if err := systemctl.ListUnits(mgr); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "list-unit-files":
		if err := systemctl.ListUnitFiles(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	case "is-system-running":
		os.Exit(systemctl.IsSystemRunning())
	case "preset-all":
		if !ctx.Quiet {
			fmt.Println("preset-all: no-op under SINS (no systemd preset logic)")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func runUnitCommand(ctx *systemctl.Ctx, mgr *runit.Manager, command, unitName string, flagNow, follow bool) int {
	if err := validateRunitServiceRef(unitName); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid unit name %q: %v\n", unitName, err)
		return 1
	}
	if ctx.UserMode && systemctl.UserModeBlocksMutation(command) {
		systemctl.PrintUserMutationError(command)
		return 1
	}
	switch command {
	case "show":
		if err := systemctl.Show(ctx, unitName); err != nil {
			var xe systemctl.ErrExit
			if asErrExit(err, &xe) {
				return xe.Code
			}
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
	case "cat":
		if err := systemctl.Cat(ctx, unitName); err != nil {
			var xe systemctl.ErrExit
			if asErrExit(err, &xe) {
				return xe.Code
			}
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
	case "mask":
		if err := ctx.Mask(unitName); err != nil {
			fmt.Fprintf(os.Stderr, "mask failed: %v\n", err)
			return 1
		}
		ctx.Logf("Masked %s\n", unitName)
	case "unmask":
		if err := ctx.Unmask(unitName); err != nil {
			fmt.Fprintf(os.Stderr, "unmask failed: %v\n", err)
			return 1
		}
		ctx.Logf("Unmasked %s\n", unitName)
	case "preset":
		if !ctx.Quiet {
			fmt.Printf("preset %s: no-op under SINS\n", unitName)
		}
	case "try-restart":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		if handleIsActiveInternal(mgr, unitName) {
			return actionExit(ctx, mgr, unitName, "restart", true)
		}
	case "reload-or-restart":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		if err := reloadSvc(ctx, mgr, unitName); err != nil {
			return actionExit(ctx, mgr, unitName, "restart", true)
		}
	case "try-reload-or-restart":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		if err := reloadSvc(ctx, mgr, unitName); err != nil {
			if handleIsActiveInternal(mgr, unitName) {
				return actionExit(ctx, mgr, unitName, "restart", true)
			}
		}
	case "kill":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		clean := strings.TrimSuffix(unitName, ".service")
		cmd := exec.Command("sv", "term", clean)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "sv term failed: %v\n", err)
			return 1
		}
	case "start":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Failed to start %s: Unit is masked.\n", unitName)
			return 1
		}
		return actionExit(ctx, mgr, unitName, "start", true)
	case "stop":
		return actionExit(ctx, mgr, unitName, "stop", false)
	case "restart":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		return actionExit(ctx, mgr, unitName, "restart", true)
	case "reload":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Unit %s is masked.\n", unitName)
			return 1
		}
		if err := reloadSvc(ctx, mgr, unitName); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
	case "status":
		return statusExit(ctx, mgr, unitName, follow)
	case "is-active":
		return isActiveExit(mgr, unitName)
	case "is-enabled":
		return isEnabledExit(mgr, unitName)
	case "enable":
		if ctx.IsMasked(unitName) {
			fmt.Fprintf(os.Stderr, "Failed to enable %s: Unit is masked.\n", unitName)
			return 1
		}
		if code := enableExit(ctx, mgr, unitName); code != 0 {
			return code
		}
		if flagNow {
			return actionExit(ctx, mgr, unitName, "start", true)
		}
	case "disable":
		if flagNow {
			if code := actionExit(ctx, mgr, unitName, "stop", false); code != 0 {
				return code
			}
		}
		return disableExit(mgr, unitName)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		return 1
	}
	return 0
}

func printUsage() {
	fmt.Println("Usage: systemctl <command> [units...] [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  start, stop, restart, try-restart, reload, reload-or-restart, try-reload-or-restart")
	fmt.Println("  status, enable, disable, mask, unmask, kill")
	fmt.Println("  is-active, is-enabled, daemon-reload")
	fmt.Println("  show, cat, list-units, list-unit-files, is-system-running")
	fmt.Println("  preset, preset-all (no-op)")
	fmt.Println("\nFlags: --user, --system, --now, --quiet/-q, -f/--follow")
}

func ensureServiceExists(ctx *systemctl.Ctx, mgr *runit.Manager, name string) {
	cleanName := strings.TrimSuffix(name, ".service")
	svPath := filepath.Join(mgr.ServiceDir, cleanName)

	if _, err := os.Stat(svPath); os.IsNotExist(err) {
		err := units.DetectCycles(name, func(n string) (*units.Unit, error) {
			path := ctx.FindUnitFile(n)
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

		if strings.Contains(cleanName, "@") {
			parts := strings.SplitN(cleanName, "@", 2)
			templateName := parts[0] + "@.service"
			instance := parts[1]

			unitPath := ctx.FindUnitFile(templateName)
			if unitPath != "" {
				var err error
				unit, err = units.Parse(unitPath)
				if err == nil {
					unit.ReplacePlaceholders(instance)
				}
			}
		}

		var parseErr error
		if unit == nil {
			unitPath := ctx.FindUnitFile(name)
			if unitPath == "" {
				fmt.Printf("Unit %s not found\n", name)
				os.Exit(1)
			}
			unit, parseErr = units.Parse(unitPath)
		}

		if parseErr != nil {
			fmt.Printf("Failed to handle unit: %v\n", parseErr)
			os.Exit(1)
		}

		if err := mgr.SetupService(cleanName, unit); err != nil {
			fmt.Printf("Failed to setup runit service: %v\n", err)
			os.Exit(1)
		}
	}
}

func actionExit(ctx *systemctl.Ctx, mgr *runit.Manager, name string, svCmd string, autoCreate bool) int {
	cleanName := strings.TrimSuffix(name, ".service")

	if autoCreate {
		ensureServiceExists(ctx, mgr, name)
	}

	if svCmd == "start" || svCmd == "restart" {
		if err := mgr.EnableService(cleanName); err != nil {
			fmt.Printf("Warning: failed to enable service %s: %v\n", cleanName, err)
		}
	}

	if err := mgr.WaitForService(cleanName, 5); err != nil {
		fmt.Printf("Warning: %v. Runit might still be initializing the service.\n", err)
	}

	cmd := exec.Command("sv", svCmd, cleanName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to execute sv %s: %v\n", svCmd, err)
		return 1
	}
	return 0
}

func reloadSvc(ctx *systemctl.Ctx, mgr *runit.Manager, name string) error {
	cleanName := strings.TrimSuffix(name, ".service")

	cmd := exec.Command("sv", "status", cleanName)
	output, _ := cmd.CombinedOutput()
	if !strings.HasPrefix(string(output), "run:") {
		return fmt.Errorf("service %s is not running, cannot reload", name)
	}

	unitPath := ctx.FindUnitFile(name)
	if unitPath != "" {
		unit, err := units.Parse(unitPath)
		if err == nil {
			execReload := unit.GetServiceField("ExecReload")
			if execReload != "" {
				fmt.Printf("Executing ExecReload: %s\n", execReload)
				reloadCmd := exec.Command("sh", "-c", units.ShellEscapeSingleQuoted(execReload))
				reloadCmd.Stdout = os.Stdout
				reloadCmd.Stderr = os.Stderr
				if err := reloadCmd.Run(); err != nil {
					return fmt.Errorf("ExecReload failed: %w", err)
				}
				return nil
			}
		}
	}

	fmt.Printf("No ExecReload found, sending SIGHUP via sv hup %s\n", cleanName)
	if err := exec.Command("sv", "hup", cleanName).Run(); err != nil {
		return fmt.Errorf("failed to send SIGHUP: %w", err)
	}
	return nil
}

func statusExit(ctx *systemctl.Ctx, mgr *runit.Manager, name string, follow bool) int {
	return systemctl.PrintServiceStatus(ctx, mgr, name, follow)
}

func isActiveExit(mgr *runit.Manager, name string) int {
	if handleIsActiveInternal(mgr, name) {
		fmt.Println("active")
		return 0
	}

	cleanName := strings.TrimSuffix(name, ".service")
	enablePath := filepath.Join(mgr.EnableDir, cleanName)
	if _, err := os.Lstat(enablePath); err == nil {
		fmt.Println("inactive (trying to start...)")
		_ = exec.Command("sv", "start", cleanName).Run()
		if handleIsActiveInternal(mgr, name) {
			fmt.Println("active")
			return 0
		}
	}

	fmt.Println("inactive")
	return 3
}

func handleIsActiveInternal(mgr *runit.Manager, name string) bool {
	cleanName := strings.TrimSuffix(name, ".service")
	cmd := exec.Command("sv", "status", cleanName)
	output, _ := cmd.CombinedOutput()
	return strings.HasPrefix(string(output), "run:")
}

func isEnabledExit(mgr *runit.Manager, name string) int {
	cleanName := strings.TrimSuffix(name, ".service")
	enablePath := filepath.Join(mgr.EnableDir, cleanName)
	if _, err := os.Lstat(enablePath); err == nil {
		fmt.Println("enabled")
		return 0
	}
	fmt.Println("disabled")
	return 1
}

func enableExit(ctx *systemctl.Ctx, mgr *runit.Manager, name string) int {
	ensureServiceExists(ctx, mgr, name)
	cleanName := strings.TrimSuffix(name, ".service")
	if err := mgr.EnableService(cleanName); err != nil {
		fmt.Printf("Failed to enable service: %v\n", err)
		return 1
	}
	fmt.Printf("Created symlink for %s\n", cleanName)
	return 0
}

func disableExit(mgr *runit.Manager, name string) int {
	cleanName := strings.TrimSuffix(name, ".service")
	if err := mgr.DisableService(cleanName); err != nil {
		fmt.Printf("Failed to disable service: %v\n", err)
		return 1
	}
	fmt.Printf("Removed symlink for %s\n", cleanName)
	return 0
}
