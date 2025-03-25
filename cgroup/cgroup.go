package cgroup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath            = "/io/hass/os/CGroup"
	ifaceName             = "io.hass.os.CGroup"
	cgroupFSDockerDevices = "/sys/fs/cgroup/devices/docker"
)

type CGroupVersion int

const (
	CGroupUnknown CGroupVersion = iota
	CGroupV1
	CGroupV2
)

type cgroup struct {
	conn          *dbus.Conn
	cgroupVersion CGroupVersion
}

func (d cgroup) AddDevicesAllowed(containerID string, permission string) (bool, *dbus.Error) {
	if d.cgroupVersion == CGroupV2 {
		permissions := []string{permission}
		resources, err := CreateDeviceUpdateResources(permissions)
		if err != nil {
			error := fmt.Errorf("creating device resources for '%s' failed: %w", containerID, err)
			logging.Error.Printf("%s", error)
			return false, dbus.MakeFailedError(error)
		}

		cmd := exec.Command("runc", "--root", "/var/run/docker/runtime-runc/moby/", "update", "--resources", "-", containerID)

		// Pass resources as OCI LinuxResources JSON object
		stdin, err := cmd.StdinPipe()
		if err != nil {
			error := fmt.Errorf("creating stdin pipe for '%s' failed: %w", containerID, err)
			logging.Error.Printf("%s", error)
			return false, dbus.MakeFailedError(error)
		}
		enc := json.NewEncoder(stdin)
		err = enc.Encode(resources)
		if err != nil {
			error := fmt.Errorf("encoding JSON for '%s' failed: %w", containerID, err)
			logging.Error.Printf("%s", error)
			return false, dbus.MakeFailedError(error)
		}
		stdin.Close()

		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			error := fmt.Errorf("calling runc for '%s' failed: %w, output %s", containerID, err, stdoutStderr)
			logging.Error.Printf("%s", error)
			return false, dbus.MakeFailedError(error)
		} else {
			logging.Info.Printf("Successfully called runc for '%s', output %s", containerID, stdoutStderr)
		}

		logging.Info.Printf("Permission '%s', granted for Container '%s' via runc", permission, containerID)
		return true, nil
	} else {
		// Make sure path is relative to cgroupFSDockerDevices
		allowedFile, err := securejoin.SecureJoin(cgroupFSDockerDevices, containerID+string(filepath.Separator)+"devices.allow")
		if err != nil {
			return false, dbus.MakeFailedError(fmt.Errorf("security issues with '%s': %w", containerID, err))
		}

		// Check if file/container exists
		_, err = os.Stat(allowedFile)
		if os.IsNotExist(err) {
			return false, dbus.MakeFailedError(fmt.Errorf("can't find Container '%s' for adjust CGroup devices", containerID))
		}

		// Write permission adjustments
		file, err := os.Create(allowedFile)
		if err != nil {
			return false, dbus.MakeFailedError(fmt.Errorf("can't open CGroup devices '%s': %w", allowedFile, err))
		}
		defer file.Close()

		_, err = file.WriteString(permission + "\n")
		if err != nil {
			return false, dbus.MakeFailedError(fmt.Errorf("can't write CGroup permission '%s': %w", permission, err))
		}

		logging.Info.Printf("Permission '%s', granted for Container '%s' via CGroup devices.allow", permission, containerID)
		return true, nil
	}
}

func InitializeDBus(conn *dbus.Conn) {
	d := cgroup{
		conn:          conn,
		cgroupVersion: CGroupUnknown,
	}

	// Check for CGroups v2
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		d.cgroupVersion = CGroupV2
		logging.Info.Printf("Detected CGroups Version 2")
	} else {
		d.cgroupVersion = CGroupV1
		logging.Info.Printf("Detected CGroups Version 1")
	}

	err := conn.Export(d, objectPath, ifaceName)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:    ifaceName,
				Methods: introspect.Methods(d),
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
