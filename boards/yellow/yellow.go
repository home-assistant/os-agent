package yellow

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards/Yellow"
	ifaceName  = "io.hass.os.Boards.Yellow"

	ledOff        = "off"
	ledSupervisor = "supervisor"
	ledLinux      = "linux"
)

var (
	optLEDPower     bool
	optLEDDisk      bool
	optLEDHeartbeat string
)

type yellow struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func getStatusLEDPower() bool {
	// FIXME: read current LED state out of sysfs
	return false
}

func getStatusLEDDisk() bool {
	// FIXME: read current LED state out of sysfs
	return false
}

func getStatusLEDHeartbeat() string {
	// FIXME: read current LED state out of sysfs
	return ledOff
}

func setStatusLEDPower(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Power LED to %t", c.Value)
	// FIXME: set new LED state out of sysfs
	optLEDPower = c.Value.(bool)
	return nil
}

func setStatusLEDDisk(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Disk LED to %t", c.Value)
	// FIXME: set new LED state out of sysfs
	optLEDDisk = c.Value.(bool)
	return nil
}

func setStatusLEDHeartbeat(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Heartbeat LED to %t", c.Value)
	// FIXME: set new LED state out of sysfs
	optLEDHeartbeat = c.Value.(string)
	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := yellow{
		conn: conn,
	}

	// Init base value
	optLEDPower = getStatusLEDPower()
	optLEDDisk = getStatusLEDDisk()
	optLEDHeartbeat = getStatusLEDHeartbeat()

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"PowerLED": {
				Value:    optLEDPower,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setStatusLEDPower,
			},
			"DiskLED": {
				Value:    optLEDDisk,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setStatusLEDDisk,
			},
			"HeartbeatLED": {
				Value:    optLEDHeartbeat,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setStatusLEDHeartbeat,
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
