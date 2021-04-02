package datadisk

import (
	"fmt"
	"log"
	"os"

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

func (d datadisk) ChangeDevice(newDevice string) (bool, *dbus.Error) {
	log.Printf("Request to change data disk to %s.", newDevice)

	udisks2helper := udisks2.NewUDisks2(d.conn)
	dataDevice, err := udisks2helper.GetRootDeviceFromLabel("hassos-data")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	fmt.Printf("Data partition is currently on device %s\n", *dataDevice)
	if *dataDevice == newDevice {
		return false, dbus.MakeFailedError(fmt.Errorf("Current data device \"%s\" the same as target device. Aborting.", *dataDevice))
	}

	err = udisks2helper.PartitionDeviceWithSinglePartition(newDevice, linuxDataPartitionUUID, "hassos-data-external")
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
		log.Panic(err)
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
		log.Panic(err)
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
		log.Panic(err)
	}

	log.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
