package scx_go_utils

// Requirement: This depends on the `power-profiles-daemon` service being installed.

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// Constants for dbus service and object paths.
const (
	defaultService = "net.hadess.PowerProfiles"
	defaultPath    = "/net/hadess/PowerProfiles"
	dbusGet        = "org.freedesktop.DBus.Properties.Get"
)

// dbus proxy for interacting with the PowerProfiles service.
// encapsulates a dbus connection and object to interact with system power profiles.
type PowerProfiles struct {
	conn *dbus.Conn
	obj  dbus.BusObject
}

var powerProfilesProxyInstance *PowerProfiles
var powerOnce sync.Once

// NewPowerProfiles initializes a new D-Bus proxy for the PowerProfiles service.
func newPowerProfiles() (*PowerProfiles, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %v", err)
	}

	obj := conn.Object(
		defaultService,
		dbus.ObjectPath(defaultPath),
	)

	return &PowerProfiles{conn, obj}, nil
}

// ActiveProfile retrieves the current active power profile from the system.
func (p *PowerProfiles) activeProfile() (string, error) {
	var profile string
	err := p.obj.Call(dbusGet, 0, defaultService, "ActiveProfile").Store(&profile)
	if err != nil {
		return "", fmt.Errorf("failed to get active profile: %w", err)
	}

	return profile, nil
}

var (
	// singleton to ensure single initialization.
	powerProfiles *PowerProfiles
)

// FetchPowerProfile retrieves the active power profile using a singleton instance.
func FetchPowerProfile() string {
	powerOnce.Do(func() {
		var err error
		powerProfiles, err = newPowerProfiles()
		if err != nil {
			fmt.Println("Error initializing PowerProfiles:", err)
			powerProfiles = nil
		}
	})

	if powerProfiles == nil {
		return ""
	}

	profile, err := powerProfiles.activeProfile()
	if err != nil {
		fmt.Println("Error fetching active profile:", err)
		return ""
	}

	return profile
}
