package datadisk

import (
	"errors"
	"fmt"
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

func GetDataMount() (*mountinfo.Mountinfo, error) {

	minfo, err := mountinfo.GetMountInfo("/proc/self/mountinfo")
	if err != nil {
		logging.Warning.Print(err)
		return nil, err
	}

	for _, info := range minfo {
		if dataMount == info.MountPoint {
			return &info, nil
		}
	}
	return nil, errors.New("can't find a data mount")
}

type datadisk struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func (d datadisk) MarkDataMove() *dbus.Error {
	/* Move request marker for hassos-data.service */
	fileName := "/mnt/overlay/move-data"
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return dbus.MakeFailedError(err)
		}
		defer file.Close()
	}

	return nil
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
		return false, dbus.MakeFailedError(fmt.Errorf("current data device \"%s\" the same as target device", *dataDevice))
	}

	err = udisks2helper.PartitionDeviceWithSinglePartition(newDevice, linuxDataPartitionUUID, "hassos-data-external")
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	dbuserr := d.MarkDataMove()
	if dbuserr != nil {
		return false, dbuserr
	}

	return true, nil
}

func (d datadisk) ReloadDevice() (bool, *dbus.Error) {
	mountInfo, err := GetDataMount()
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}

	d.props.SetMust(ifaceName, "CurrentDevice", mountInfo.MountSource)
	return true, nil
}

const (
	objectPath = "/io/hass/os/DataDisk"
	ifaceName  = "io.hass.os.DataDisk"
)

func InitializeDBus(conn *dbus.Conn) {

	// Try to read the current data mount point
	mountInfo, err := GetDataMount()
	var currentDisk = ""
	if err != nil {
		logging.Warning.Print(err)
	} else {
		currentDisk = mountInfo.MountSource
	}

	d := datadisk{
		conn: conn,
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"CurrentDevice": {
				Value:    &currentDisk,
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
	d.props = props

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
