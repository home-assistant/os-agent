package cgroup

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath            = "/io/homeassistant/os/CGroup"
	ifaceName             = "io.homeassistant.os.CGroup"
	CGroupFSDockerDevices = "/sys/fs/cgroup/devices/docker/"
)

type cgroup struct {
	conn *dbus.Conn
}

func (d cgroup) UpdateDevicesAllowed(containerID string, permission string) (bool, *dbus.Error) {
	allowedFile := CGroupFSDockerDevices + containerID + "/devices.allow"

	// Check if file/container exists
	_, err := os.Stat(allowedFile)
	if os.IsNotExist(err) {
		return false, dbus.MakeFailedError(fmt.Errorf("Can't find Container '%s' for adjust CGroup devices.", containerID))
	}

	// Write permission adjustments
	file, err := os.Create(allowedFile)
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("Can't open CGroup devices '%s': %s", allowedFile, err))
	}
	defer file.Close()

	_, err = file.WriteString(permission + "\n")
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("Can't write CGroup permission '%s': %s", permission, err))
	}

	logging.Info.Printf("Permission '%s', granted for Container '%s'", containerID, permission)
	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := cgroup{
		conn: conn,
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
