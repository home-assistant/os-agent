package main

import (
	os_time "github.com/home-assistant/os-agent/time"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/getsentry/sentry-go"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/apparmor"
	"github.com/home-assistant/os-agent/boards"
	"github.com/home-assistant/os-agent/cgroup"
	"github.com/home-assistant/os-agent/datadisk"
	"github.com/home-assistant/os-agent/system"
	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	busName    = "io.hass.os"
	objectPath = "/io/hass/os"
	sentryDsn  = "https://c74e811a96e4413a95caaaa5ae05f851@o427061.ingest.sentry.io/5710878"
)

var (
	// version and baord are set at link time via -X flag
	version       string = "dev"
	board         string = "unknown"
	enableCapture bool   = false
)

func main() {
	logging.Info.Printf("Start OS-Agent %s", version)

	// Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:        sentryDsn,
		Release:    version,
		BeforeSend: filterSentry,
	})
	if err != nil {
		logging.Critical.Fatalf("Sentry init: %s", err)
	}

	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	// Connect DBus
	conn, err := dbus.SystemBus()
	if err != nil {
		logging.Critical.Fatalf("DBus connection: %s", err)
	}

	// Init Dbus io.hass.os
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		logging.Critical.Panic(err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		logging.Critical.Fatalf("name already taken")
	}

	// Set base Property / functionality
	InitializeDBus(conn)

	logging.Info.Printf("Listening on service %s ...", busName)
	datadisk.InitializeDBus(conn)
	system.InitializeDBus(conn)
	apparmor.InitializeDBus(conn)
	cgroup.InitializeDBus(conn)
	boards.InitializeDBus(conn, board)
	os_time.InitializeDBus(conn)

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
			"Diagnostics": {
				Value:    enableCapture,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: func(c *prop.Change) *dbus.Error {
					logging.Info.Printf("Diagnostics is now %t", c.Value)
					enableCapture = c.Value.(bool)
					return nil
				},
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
