package apparmor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath        = "/io/hass/os/AppArmor"
	ifaceName         = "io.hass.os.AppArmor"
	appArmorParserCmd = "apparmor_parser"
)

type apparmor struct {
	conn  *dbus.Conn
	props *prop.Properties
}

func getAppArmorVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, appArmorParserCmd, "--version")

	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Warning.Print(err)
		return string("")
	}

	re := regexp.MustCompile("version ([0-9.]*)")
	found := re.FindSubmatch(out)
	if len(found) < 1 {
		logging.Error.Fatalln("Can't read version from parser!")
	}

	return string(found[1])
}

// profileNames returns the names of all profiles defined in the given file as
// reported by the parser, one entry per profile. Child profiles and hats are
// reported as "parent//child".
func profileNames(ctx context.Context, profilePath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, appArmorParserCmd, "--names", "--skip-cache", profilePath)
	out, err := cmd.Output()
	if err != nil {
		// The last line of stderr carries the actual parser error; earlier
		// lines are warnings such as a missing cache interface.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.Split(strings.TrimSpace(string(exitErr.Stderr)), "\n")
			if last := strings.TrimSpace(stderr[len(stderr)-1]); last != "" {
				return nil, errors.New(last)
			}
		}
		return nil, err
	}

	var names []string
	for _, name := range strings.Split(string(out), "\n") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// checkProfileNames verifies that a profile file only defines the profile
// named after the file itself, plus child profiles and hats of that profile.
//
// The parser loads (and with --replace, redefines) every profile found in a
// file regardless of its name. Callers store each profile in a file named
// after it, so anything else in the file would replace an unrelated profile,
// for example docker-default or the Supervisor's own profile.
func checkProfileNames(profilePath string, names []string) error {
	expected := filepath.Base(profilePath)
	found := false

	for _, name := range names {
		switch {
		case name == expected:
			found = true
		case strings.HasPrefix(name, expected+"//"):
			// child profile or hat of the expected profile
		default:
			return fmt.Errorf("profile file '%s' defines unexpected profile '%s'", profilePath, name)
		}
	}

	if !found {
		return fmt.Errorf("profile file '%s' does not define profile '%s'", profilePath, expected)
	}
	return nil
}

func validateProfile(ctx context.Context, profilePath string) error {
	names, err := profileNames(ctx, profilePath)
	if err != nil {
		return fmt.Errorf("can't parse profile '%s': %w", profilePath, err)
	}
	return checkProfileNames(profilePath, names)
}

func (d apparmor) LoadProfile(profilePath string, cachePath string) (bool, *dbus.Error) {
	logging.Info.Printf("Load AppArmor profile '%s'.", profilePath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := validateProfile(ctx, profilePath); err != nil {
		return false, dbus.MakeFailedError(err)
	}

	cmd := exec.CommandContext(ctx, appArmorParserCmd, "--replace", "--write-cache", "--cache-loc", cachePath, profilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("can't load profile '%s': %w", profilePath, err))
	}

	logging.Info.Printf("Load profile '%s': %s", profilePath, out)
	return true, nil
}

func (d apparmor) UnloadProfile(profilePath string, cachePath string) (bool, *dbus.Error) {
	logging.Info.Printf("Unload AppArmor profile '%s'.", profilePath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := validateProfile(ctx, profilePath); err != nil {
		return false, dbus.MakeFailedError(err)
	}

	cmd := exec.CommandContext(ctx, appArmorParserCmd, "--remove", "--write-cache", "--cache-loc", cachePath, profilePath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, dbus.MakeFailedError(fmt.Errorf("can't unload profile '%s': %w", profilePath, err))
	}

	logging.Info.Printf("Unload profile '%s': %s", profilePath, out)
	return true, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := apparmor{
		conn: conn,
	}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"ParserVersion": {
				Value:    getAppArmorVersion(),
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
