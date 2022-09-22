package yellow

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/home-assistant/os-agent/utils/bootfile"
	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards/Yellow"
	ifaceName  = "io.hass.os.Boards.Yellow"
	bootConfig = "/mnt/boot/config.txt"
)

var (
	optLEDPower     bool
	optLEDDisk      bool
	optLEDHeartbeat bool
	bootFile        = bootfile.Editor{FilePath: bootConfig, Delimiter: "="}
)

type yellow struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func getStatusLEDPower() bool {
	value, _ := bootFile.ReadOption("dtparam=pwr_led_trigger", "default-on")
	return value != "none"
}

func getStatusLEDDisk() bool {
	value, _ := bootFile.ReadOption("dtparam=act_led_trigger", "activity")
	return value != "none"
}

func getStatusLEDHeartbeat() bool {
	value, _ := bootFile.ReadOption("dtparam=usr_led_trigger", "heartbeat")
	return value != "none"
}

func setStatusLEDPower(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Power LED to %t", c.Value)
	optLEDPower = c.Value.(bool)

	var err error
	if c.Value.(bool) {
		err = bootFile.DisableOption("dtparam=pwr_led_trigger")
	} else {
		err = bootFile.SetOption("dtparam=pwr_led_trigger", "none")
	}

	if err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func setStatusLEDDisk(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Disk LED to %t", c.Value)
	optLEDDisk = c.Value.(bool)

	var err error
	if c.Value.(bool) {
		err = bootFile.DisableOption("dtparam=act_led_trigger")
	} else {
		err = bootFile.SetOption("dtparam=act_led_trigger", "none")
	}

	if err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func setStatusLEDHeartbeat(c *prop.Change) *dbus.Error {
	logging.Info.Printf("Set Yellow Heartbeat LED to %t", c.Value)
	optLEDHeartbeat = c.Value.(bool)

	var err error
	if c.Value.(bool) {
		err = bootFile.DisableOption("dtparam=usr_led_trigger")
	} else {
		err = bootFile.SetOption("dtparam=usr_led_trigger", "none")
	}

	if err != nil {
		return dbus.MakeFailedError(err)
	}
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
