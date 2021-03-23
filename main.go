package main

import (
	"fmt"
	"os"

	"github.com/home-assistant/os-agent/datadisk"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/godbus/dbus/v5"
)

const (
	busName = "io.homeassistant.os"
)

func main() {

	conn, err := dbus.SystemBus()
	if err != nil {
		panic(err)
	}

	reply, err := conn.RequestName(busName,
		dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		fmt.Fprintln(os.Stderr, "name already taken")
		os.Exit(1)
	}

	fmt.Printf("Listening on service %s ...\n", busName)
	datadisk.InitializeDBus(conn)

	daemon.SdNotify(false, daemon.SdNotifyReady)
	select {}
}
