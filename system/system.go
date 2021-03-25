package system

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/home-assistant/os-agent/udisks2"
)

const (
	objectPath             = "/io/homeassistant/os/System"
	ifaceName              = "io.homeassistant.os.System"
	labelDataFileSystem    = "hassos-data"
	labelOverlayFileSystem = "hassos-overlay"
)

type system struct {
	conn *dbus.Conn
}

func getAndCheckBusObjectFromLabel(udisks2helper udisks2.UDisks2Helper, label string) (dbus.BusObject, error) {
	dataBusObject, err := udisks2helper.GetBusObjectFromLabel(label)
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	dataFilesystem := udisks2.NewFilesystem(dataBusObject)
	dataMountPoints, err := dataFilesystem.GetMountPointsString(context.Background())
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}

	if len(dataMountPoints) > 0 {
		return nil, dbus.MakeFailedError(fmt.Errorf("Device with label \"%s\" is mounted at %s, aborting.", label, dataMountPoints))
	}

	return dataBusObject, nil
}

func (d system) FactoryReset() (bool, *dbus.Error) {
	fmt.Printf("Wipe device data.\n")

	udisks2helper := udisks2.NewUDisks2(d.conn)
	dataBusObject, err := getAndCheckBusObjectFromLabel(udisks2helper, labelDataFileSystem)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	overlayBusObject, err := getAndCheckBusObjectFromLabel(udisks2helper, labelOverlayFileSystem)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	err = udisks2helper.FormatPartition(dataBusObject, "ext4", labelDataFileSystem)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	err = udisks2helper.FormatPartition(overlayBusObject, "ext4", labelOverlayFileSystem)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	fmt.Printf("Successfully wiped device data.\n")

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
