package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/home-assistant/os-agent/udisks2"

	"github.com/fntlnz/mountinfo"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	dataMount              = "/mnt/data"
	linuxDataPartitionUUID = "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
)

func GetDataMount() *mountinfo.Mountinfo {

	minfo, err := mountinfo.GetMountInfo("/proc/self/mountinfo")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	for _, info := range minfo {
		if dataMount == info.MountPoint {
			return &info
		}
	}
	return nil
}

type datadisk struct {
	currentDisk string
	conn        *dbus.Conn
}

var noOptions = map[string]dbus.Variant{}

func GetDataDevice(conn *dbus.Conn, m *udisks2.Manager) (dataDevice *string, err error) {
	devspec := map[string]dbus.Variant{"label": dbus.MakeVariant("hassos-data")}
	blockObjects, err := m.ResolveDevice(context.Background(), devspec, noOptions)

	if len(blockObjects) != 1 {
		return nil, fmt.Errorf("Expected single block device with file system label \"hassos-data\", found %d", len(blockObjects))
	}

	/* Get Partition object of the data partition */
	busObjectBlock := conn.Object("org.freedesktop.UDisks2", blockObjects[0])
	partition := udisks2.NewPartition(busObjectBlock)
	table, err := partition.GetTable(context.Background())
	if err != nil {
		return nil, err
	}

	/* Get Block device of partition table */
	busObjectParentBlock := conn.Object("org.freedesktop.UDisks2", table)
	parentBlock := udisks2.NewBlock(busObjectParentBlock)

	device, err := parentBlock.GetDevice(context.Background())
	if err != nil {
		return nil, err
	}

	s := strings.Trim(string(device), "\x00")
	return &s, nil
}

func FormatDevice(conn *dbus.Conn, m *udisks2.Manager, devicePath string) (err error) {
	devspec := map[string]dbus.Variant{"path": dbus.MakeVariant(devicePath)}
	blockObjects, err := m.ResolveDevice(context.Background(), devspec, noOptions)

	if len(blockObjects) != 1 {
		return fmt.Errorf("Expected single block device with device path \"%s\", found %d", devicePath, len(blockObjects))
	}

	blockObjectPath := blockObjects[0]
	fmt.Printf("Formatting device %s\n", devicePath)

	busObjectParentBlock := conn.Object("org.freedesktop.UDisks2", blockObjectPath)
	parentBlock := udisks2.NewBlock(busObjectParentBlock)
	err = parentBlock.Format(context.Background(), "gpt", noOptions)
	if err != nil {
		return err
	}

	parentPartitionTable := udisks2.NewPartitionTable(busObjectParentBlock)
	createdPartition, err :=
		parentPartitionTable.CreatePartition(context.Background(), 0, 0,
			linuxDataPartitionUUID, "hassos-data-external", noOptions)
	if err != nil {
		return err
	}
	fmt.Printf("New part %s\n", createdPartition)

	return nil
}

func (d datadisk) ChangeDevice(newDevice string) (bool, *dbus.Error) {
	fmt.Printf("Request to change data disk to %s.\n", newDevice)

	busObj := d.conn.Object("org.freedesktop.UDisks2", "/org/freedesktop/UDisks2/Manager")
	manager := udisks2.NewManager(busObj)
	dataDevice, err := GetDataDevice(d.conn, manager)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	fmt.Printf("Data partition is currently on device %s\n", *dataDevice)
	if *dataDevice == newDevice {
		return false, dbus.MakeFailedError(fmt.Errorf("Current data device \"%s\" the same as target device. Aborting.", *dataDevice))
	}

	err = FormatDevice(d.conn, manager, newDevice)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	/* Move request marker for hassos-data.service */
	fileName := "/mnt/overlay/move-data"
	_, err = os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return false, dbus.MakeFailedError(err)
		}
		defer file.Close()
	}

	return true, nil
}

const (
	objectPath = "/io/homeassistant/os/DataDisk"
	ifaceName  = "io.homeassistant.os.DataDisk"
)

func InitializeDBus(conn *dbus.Conn) {

	/* Since we don't remount data disk at runtime, we can assume the current value remains  */
	mountInfo := GetDataMount()
	var currentDisk string = ""
	if mountInfo != nil {
		currentDisk = mountInfo.MountSource
	}

	d := datadisk{
		currentDisk: currentDisk,
		conn:        conn,
	}

	err := conn.Export(d, objectPath, ifaceName)
	if err != nil {
		panic(err)
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"CurrentDevice": {
				Value:    &d.currentDisk,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
		},
	}
	props, err := prop.Export(conn, objectPath, propsSpec)
	if err != nil {
		panic(err)
	}

	node := &introspect.Node{}
	node.Name = ifaceName
	iface := &introspect.Interface{}

	iface.Name = ifaceName

	mts := introspect.Methods(d)
	iface.Methods = mts
	iface.Properties = props.Introspection(ifaceName)

	node.Interfaces = append(node.Interfaces, *iface)

	dbus_xml_str := introspect.NewIntrospectable(node)
	err = conn.Export(dbus_xml_str, objectPath,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Exposing object %s with interface %s ...\n", objectPath, ifaceName)
}
