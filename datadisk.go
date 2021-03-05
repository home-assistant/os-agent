package main

import (
	"fmt"
	"time"

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
		fmt.Println(err.Error)
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

func (d datadisk) ChangeDataDisk(newDevice string) (bool, *dbus.Error) {
	time.Sleep(5 * time.Second)
	fmt.Println(newDevice)
	return true, nil
}

const (
	objectPath = "/io/homeassistant/os/DataDisk"
	ifaceName  = "io.homeassistant.os.DataDisk"
)

func InitializeDBus(conn *dbus.Conn) {

	d := datadisk{
		currentDisk: "",
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
				Callback: func(c *prop.Change) *dbus.Error {
					fmt.Println(c.Name, "changed to", c.Value)
					return nil
				},
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
