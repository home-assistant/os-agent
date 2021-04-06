package system

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/udisks2"
	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath             = "/io/homeassistant/os/System"
	ifaceName              = "io.homeassistant.os.System"
	labelDataFileSystem    = "hassos-data"
	labelOverlayFileSystem = "hassos-overlay"
	kernelCommandLine      = "/mnt/boot/cmdline.txt"
	tmpKernelCommandLine   = "/mnt/boot/.tmp.cmdline.txt"
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

func (d system) WipeDevice() (bool, *dbus.Error) {
	logging.Info.Printf("Wipe device data.")

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
	logging.Info.Printf("Successfully wiped device data.")

	return true, nil
}

func (d system) ScheduleWipeDevice() (bool, *dbus.Error) {

	data, err := ioutil.ReadFile(kernelCommandLine)
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	datastr := strings.TrimSpace(string(data))
	datastr += " haos.wipe=1"

	err = ioutil.WriteFile(tmpKernelCommandLine, []byte(datastr), 0644)
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	// Boot is mounted sync on Home Assistant OS, so just rename should be fine.
	err = os.Rename(tmpKernelCommandLine, kernelCommandLine)
	if err != nil {
		fmt.Println(err)
		return false, dbus.MakeFailedError(err)
	}

	logging.Info.Printf("Device will get wiped on next reboot!")
	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := system{
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
