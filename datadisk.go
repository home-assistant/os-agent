package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fntlnz/mountinfo"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	dataMount = "/mnt/data"
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
}

func (d datadisk) ChangeDevice(newDevice string) (bool, *dbus.Error) {
	fmt.Printf("Changing data disk to %s\n", newDevice)
	cmd := exec.Command("/usr/bin/datactl", "move", newDevice)
	cmd.Env = append(os.Environ(),
		"DATACTL_NOCONFIRM=1",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	fmt.Print(out.String())
	if err != nil {
		fmt.Println(err)
		return false, nil
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
