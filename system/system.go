package system

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/home-assistant/os-agent/udisks2"

	systemddbus "github.com/coreos/go-systemd/v22/dbus"
)

const (
	objectPath = "/io/homeassistant/os/System"
	ifaceName  = "io.homeassistant.os.System"
)

type system struct {
	conn *dbus.Conn
}

func SystemdIsolate(c *systemddbus.Conn, target string) error {
	result := make(chan string, 1) // catch result information
	_, err := c.StartUnitContext(context.Background(), target, "isolate", result)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("Isolating haos-maintenance.target failed: Result is nil")
	}

	status := <-result
	if status != "done" {
		return fmt.Errorf("Isolating haos-maintenance.target failed: Unknown return string: %s", status)
	}

	return nil
}

func (d system) FactoryReset() (bool, *dbus.Error) {
	fmt.Printf("Wipe device data.\n")

	c, err := systemddbus.NewSystemConnection()
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	states, err := c.ListUnitsByNames([]string{"default.target"})
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	if len(states) < 1 {
		return false, dbus.MakeFailedError(fmt.Errorf("Unit \"default.target\" not found."))
	}

	fmt.Printf("State of \"default.target\" is currently %s\n", states[0].ActiveState)

	err = SystemdIsolate(c, "haos-maintenance.target")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	udisks2helper := udisks2.NewUDisks2(d.conn)
	dataDevice, err := udisks2helper.GetPartitionDeviceFromLabel("hassos-data")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	overlayDevice, err := udisks2helper.GetPartitionDeviceFromLabel("hassos-overlay")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	err = udisks2helper.FormatPartition(*dataDevice, "ext4", "hassos-data")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	err = udisks2helper.FormatPartition(*overlayDevice, "ext4", "hassos-overlay")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	fmt.Printf("Successfully wiped device data.\n")

	err = SystemdIsolate(c, "default.target")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := system{
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

	fmt.Printf("Exposing object %s with interface %s ...\n", objectPath, ifaceName)
}
