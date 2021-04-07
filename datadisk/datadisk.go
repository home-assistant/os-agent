package datadisk

import (
	"fmt"
	"log"
	"os"

	"github.com/fntlnz/mountinfo"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/udisks2"
	logging "github.com/home-assistant/os-agent/utils/log"
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
	logging.Info.Printf("Request to change data disk to %s.", newDevice)

	udisks2helper := udisks2.NewUDisks2(d.conn)
	dataDevice, err := udisks2helper.GetRootDeviceFromLabel("hassos-data")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	logging.Info.Printf("Data partition is currently on device %s.", *dataDevice)
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
		logging.Critical.Panic(err)
	}

	err = conn.Export(d, objectPath, ifaceName)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       ifaceName,
				Methods:    introspect.Methods(d),
				Properties: props.Introspection(ifaceName),
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
