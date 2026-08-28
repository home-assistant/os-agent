package timesyncd

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/home-assistant/os-agent/utils/lineinfile"
	"regexp"
	"strings"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath    = "/io/hass/os/Config/Timesyncd"
	ifaceName     = "io.hass.os.Config.Timesyncd"
	timesyncdConf = "/etc/systemd/timesyncd.conf"
)

var (
	optNTPServer         []string
	optFallbackNTPServer []string
	configFile           = lineinfile.LineInFile{FilePath: timesyncdConf}
)

type timesyncd struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func getNTPServers() []string {
	return getTimesyncdConfigProperty("NTP")
}

func getFallbackNTPServers() []string {
	return getTimesyncdConfigProperty("FallbackNTP")
}

func setNTPServer(c *prop.Change) *dbus.Error {
	servers, ok := c.Value.([]string)
	if !ok {
		return dbus.MakeFailedError(fmt.Errorf("invalid type for NTPServer"))
	}

	if err := updateTimesyncdConfigProperty("NTP", servers); err != nil {
		return dbus.MakeFailedError(err)
	}

	optNTPServer = servers
	return nil
}

func setFallbackNTPServer(c *prop.Change) *dbus.Error {
	servers, ok := c.Value.([]string)
	if !ok {
		return dbus.MakeFailedError(fmt.Errorf("invalid type for FallbackNTPServer"))
	}

	if err := updateTimesyncdConfigProperty("FallbackNTP", servers); err != nil {
		return dbus.MakeFailedError(err)
	}

	optFallbackNTPServer = servers
	return nil
}

func getTimesyncdConfigProperty(property string) []string {
	value, err := configFile.Find(`^\s*(`+property+`=).*$`, `\[Time\]`, true)

	var servers []string

	if err != nil || value == nil {
		return servers
	}

	matches := regexp.MustCompile(property + `=([^\s#]+(?:\s+[^\s#]+)*)`).FindStringSubmatch(*value)
	if len(matches) > 1 {
		servers = strings.Split(matches[1], " ")
	}

	return servers
}

// updateTimesyncdConfigProperty removes the property when servers is empty, so
// lower-priority configuration (e.g. servers from DHCP) applies again.
func updateTimesyncdConfigProperty(property string, servers []string) error {
	if len(servers) == 0 {
		return unsetTimesyncdConfigProperty(property)
	}
	return setTimesyncdConfigProperty(property, strings.Join(servers, " "))
}

func unsetTimesyncdConfigProperty(property string) error {
	var params = lineinfile.NewAbsentParams()
	params.Regexp, _ = regexp.Compile(`^\s*(` + property + `=).*$`)
	params.After = `\[Time\]`
	if err := configFile.Absent(params); err != nil {
		return fmt.Errorf("failed to unset %s: %w", property, err)
	}

	return nil
}

func setTimesyncdConfigProperty(property string, value string) error {
	var params = lineinfile.NewPresentParams(property + "=" + value)
	params.Regexp, _ = regexp.Compile(`^\s*#?\s*(` + property + `=).*$`)
	// Keep it simple, timesyncd.conf only has the [Time] section
	params.After = `\[Time\]`
	if err := configFile.Present(params); err != nil {
		return fmt.Errorf("failed to set %s: %w", property, err)
	}

	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := timesyncd{
		conn: conn,
	}

	optNTPServer = getNTPServers()
	optFallbackNTPServer = getFallbackNTPServers()

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"NTPServer": {
				Value:    optNTPServer,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setNTPServer,
			},
			"FallbackNTPServer": {
				Value:    optFallbackNTPServer,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: setFallbackNTPServer,
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
