package cgroup

import (
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
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

	log.Printf("Permission '%s', granted for Container '%s'", containerID, permission)
	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := cgroup{
		conn: conn,
	}

	err := conn.Export(d, objectPath, ifaceName)
	if err != nil {
		panic(err)
	}

	node := &introspect.Node{}
	node.Name = ifaceName
	iface := &introspect.Interface{}

	iface.Name = ifaceName

	mts := introspect.Methods(d)
	iface.Methods = mts

	node.Interfaces = append(node.Interfaces, *iface)

	dbus_xml_str := introspect.NewIntrospectable(node)
	err = conn.Export(dbus_xml_str, objectPath,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		panic(err)
	}

	log.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
