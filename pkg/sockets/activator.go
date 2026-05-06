package sockets

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"shim-systemctl/pkg/safeunit"
	"shim-systemctl/pkg/units"
	"strings"
)

func unixSocketModeFromEnv(key string, defaultOct uint32) os.FileMode {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return os.FileMode(defaultOct)
	}
	var u uint32
	if n, err := fmt.Sscanf(s, "%o", &u); n != 1 || err != nil || u > 07777 {
		return os.FileMode(defaultOct)
	}
	return os.FileMode(u)
}

type Activator struct {
	UnitPath    string
	ServicePath string
}

func NewActivator(unitPath, servicePath string) *Activator {
	return &Activator{
		UnitPath:    unitPath,
		ServicePath: servicePath,
	}
}

func (a *Activator) Run() error {
	unit, err := units.Parse(a.UnitPath)
	if err != nil {
		return err
	}

	listenStream := unit.Get("Socket", "ListenStream")
	if listenStream == "" {
		return fmt.Errorf("no ListenStream found in %s", a.UnitPath)
	}

	// For now, only Unix sockets are supported
	if !strings.HasPrefix(listenStream, "/") {
		return fmt.Errorf("only absolute path unix sockets are supported: %s", listenStream)
	}

	// Clean up existing socket
	os.Remove(listenStream)

	l, err := net.Listen("unix", listenStream)
	if err != nil {
		return err
	}
	defer l.Close()

	mode := unixSocketModeFromEnv("SINS_UNIX_SOCKET_MODE", 0666)
	_ = os.Chmod(listenStream, mode)

	fmt.Printf("Listening on %s\n", listenStream)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		
		fmt.Println("Connection received, starting service...")
		
		// Trigger the service
		unitFile := filepath.Base(a.UnitPath)
		serviceName := strings.TrimSuffix(unitFile, ".socket") + ".service"
		svcBase := strings.TrimSuffix(serviceName, ".service")
		if err := safeunit.ValidateServiceName(svcBase); err != nil {
			fmt.Printf("Invalid service name %q: %v\n", serviceName, err)
			conn.Close()
			continue
		}

		cmd := exec.Command(a.ServicePath, "start", serviceName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to start service: %v\n", err)
		}
		
		// Close the connection as we don't support real socket passing yet
		// This will cause the client to reconnect to the now-running service
		conn.Close()
	}
}
