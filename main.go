package main

import (
	"fmt"
	"os"

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

	fmt.Println(fmt.Sprintf("Listening on service %s ...", busName))
	InitializeDBus(conn)

	daemon.SdNotify(false, daemon.SdNotifyReady)
	select {}
}
