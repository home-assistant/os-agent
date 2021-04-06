package main

import (
	"github.com/home-assistant/os-agent/datadisk"
	"github.com/home-assistant/os-agent/system"
	logging "github.com/home-assistant/os-agent/utils/log"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/godbus/dbus/v5"
)

const (
	busName    = "io.homeassistant.os"
	objectPath = "/io/homeassistant/os"
	version    = "dev"
)

func main() {

	conn, err := dbus.SystemBus()
	if err != nil {
		logging.Critical.Panic(err)
	}

	// Init Dbus
	reply, err := conn.RequestName(busName,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		logging.Critical.Panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		logging.Critical.Panic("name already taken")
	}

	// Set base Property
	base_properties(conn)

	logging.Info.Printf("Listening on service %s ...", busName)
	datadisk.InitializeDBus(conn)
	system.InitializeDBus(conn)

	_, err = daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		logging.Critical.Panic(err)
	}
	select {}
}
