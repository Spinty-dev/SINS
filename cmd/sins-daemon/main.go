package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"shim-systemctl/pkg/dbus"
	"shim-systemctl/pkg/notify"
	"shim-systemctl/pkg/runit"

	dbus_lib "github.com/godbus/dbus/v5"
)

func main() {
	mgr := runit.NewManager()

	// Initialize Notify Listener
	nl, err := notify.NewListener(mgr.ServiceDir)
	if err != nil {
		log.Fatalf("Failed to start notify listener: %v", err)
	}
	fmt.Println("Notify listener started at /run/systemd/notify")
	go nl.Start()

	// Initialize D-Bus Bridge
	conn, err := dbus_lib.SystemBus()
	if err != nil {
		// Fallback to SessionBus for development/testing
		conn, err = dbus_lib.SessionBus()
		if err != nil {
			log.Fatalf("Failed to connect to D-Bus: %v", err)
		}
	}

	// Request org.freedesktop.systemd1 name
	reply, err := conn.RequestName("org.freedesktop.systemd1", dbus_lib.NameFlagReplaceExisting)
	if err != nil {
		fmt.Printf("Warning: Could not request name org.freedesktop.systemd1: %v\n", err)
	} else if reply != dbus_lib.RequestNameReplyPrimaryOwner {
		fmt.Println("Warning: Name org.freedesktop.systemd1 already owned by another process")
	}

	if err := dbus.Register(conn, mgr); err != nil {
		log.Fatalf("Failed to register D-Bus Manager: %v", err)
	}

	fmt.Println("SINS Daemon is running (D-Bus, Notify, Cgroups integrated)")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down SINS Daemon...")
}
