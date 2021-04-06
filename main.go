package main

import (
	"github.com/home-assistant/os-agent/apparmor"
	"github.com/home-assistant/os-agent/cgroup"
	"github.com/home-assistant/os-agent/datadisk"
	"github.com/home-assistant/os-agent/system"
	logging "github.com/home-assistant/os-agent/utils/log"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	busName    = "io.homeassistant.os"
	objectPath = "/io/homeassistant/os"
)

var version string

func main() {
	logging.Info.Printf("Start OS-Agent v%s", version)

	conn, err := dbus.SystemBus()
	if err != nil {
		logging.Critical.Panic(err)
	}

	// Init Dbus
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		logging.Critical.Panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		logging.Critical.Panic("name already taken")
	}

	// Set base Property / functionality
	InitializeDBus(conn)

	logging.Info.Printf("Listening on service %s ...", busName)
	datadisk.InitializeDBus(conn)
	system.InitializeDBus(conn)
	apparmor.InitializeDBus(conn)
	cgroup.InitializeDBus(conn)

	_, err = daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		logging.Critical.Panic(err)
	}
	select {}
}

func InitializeDBus(conn *dbus.Conn) {
	propsSpec := map[string]map[string]*prop.Prop{
		busName: {
			"Version": {
				Value:    version,
				Writable: false,
				Emit:     prop.EmitInvalidates,
				Callback: nil,
			},
		},
	}

	props, err := prop.Export(conn, objectPath, propsSpec)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       busName,
				Properties: props.Introspection(busName),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, busName)
}
