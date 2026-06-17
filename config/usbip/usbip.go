package usbip

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/natefinch/atomic"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath  = "/io/hass/os/Config/USBIP"
	ifaceName   = "io.hass.os.Config.USBIP"
	defaultPort = uint32(3240)
	configExt   = ".conf"
)

var (
	// configDir is a var (not const) so tests can redirect it to a temp dir.
	configDir = "/run/usbip/remote-devices"

	// validIdentifier restricts identifiers to a safe filename charset. A UUID
	// (the expected caller-supplied identifier) matches this pattern. Dots are
	// deliberately excluded so that "", ".", "..", path separators, and any
	// ".conf" extension can never appear in an identifier.
	validIdentifier = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

type usbip struct {
	conn *dbus.Conn
}

// validateIdentifier ensures the caller-supplied identifier is safe to use as
// a filename inside configDir, guarding against path traversal.
func validateIdentifier(identifier string) error {
	// The strict charset (no dots, slashes, or whitespace) already rules out
	// empty, ".", "..", and any path separators.
	if !validIdentifier.MatchString(identifier) {
		return fmt.Errorf("identifier %q contains invalid characters", identifier)
	}
	// Defense in depth: the resolved path must stay directly inside configDir.
	if filepath.Dir(filepath.Join(configDir, identifier)) != filepath.Clean(configDir) {
		return fmt.Errorf("invalid identifier %q", identifier)
	}
	return nil
}

// configPath returns the on-disk path of the config file for an identifier.
func configPath(identifier string) string {
	return filepath.Join(configDir, identifier+configExt)
}

// buildConfig renders the systemd EnvironmentFile content for a remote device.
func buildConfig(host, busID string, port uint32, name string) string {
	if port == 0 {
		port = defaultPort
	}

	var b strings.Builder
	fmt.Fprintf(&b, "HOST=%s\n", host)
	fmt.Fprintf(&b, "BUSID=%s\n", busID)
	fmt.Fprintf(&b, "PORT=%d\n", port)
	fmt.Fprintf(&b, "NAME=%s\n", name)
	return b.String()
}

// containsNewline reports whether any field contains a newline, which would
// corrupt the EnvironmentFile.
func containsNewline(values ...string) bool {
	for _, v := range values {
		if strings.ContainsAny(v, "\n\r") {
			return true
		}
	}
	return false
}

// WriteRemoteDevice creates or updates the config file for the given
// identifier. It is an upsert: an existing file is overwritten.
func (d usbip) WriteRemoteDevice(identifier string, host string, busID string, port uint32, name string) *dbus.Error {
	if err := validateIdentifier(identifier); err != nil {
		return dbus.MakeFailedError(err)
	}
	if host == "" {
		return dbus.MakeFailedError(fmt.Errorf("host must not be empty"))
	}
	if busID == "" {
		return dbus.MakeFailedError(fmt.Errorf("busID must not be empty"))
	}
	if containsNewline(host, busID, name) {
		return dbus.MakeFailedError(fmt.Errorf("fields must not contain newlines"))
	}
	if port > 65535 {
		return dbus.MakeFailedError(fmt.Errorf("port %d out of range (0-65535)", port))
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to create config directory: %w", err))
	}

	content := buildConfig(host, busID, port, name)
	path := configPath(identifier)
	if err := atomic.WriteFile(path, strings.NewReader(content)); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to write config for %q: %w", identifier, err))
	}

	logging.Info.Printf("Wrote usbip remote-device config %s", identifier)
	return nil
}

// RemoveRemoteDevice deletes the config file for the given identifier. Removing
// a non-existent identifier is treated as success.
func (d usbip) RemoveRemoteDevice(identifier string) *dbus.Error {
	if err := validateIdentifier(identifier); err != nil {
		return dbus.MakeFailedError(err)
	}

	path := configPath(identifier)
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return dbus.MakeFailedError(fmt.Errorf("failed to remove config for %q: %w", identifier, err))
		}
		return nil
	}

	logging.Info.Printf("Removed usbip remote-device config %s", identifier)
	return nil
}

// ListRemoteDevices returns the identifiers of all known remote-device config
// files, sorted.
func (d usbip) ListRemoteDevices() ([]string, *dbus.Error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, dbus.MakeFailedError(fmt.Errorf("failed to list configs: %w", err))
	}

	identifiers := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), configExt) {
			continue
		}
		identifier := strings.TrimSuffix(entry.Name(), configExt)
		// Only surface identifiers we would accept ourselves, keeping List()
		// consistent with the Write/Remove contract (e.g. drop a bare ".conf").
		if validateIdentifier(identifier) != nil {
			continue
		}
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	return identifiers, nil
}

func InitializeDBus(conn *dbus.Conn) {
	d := usbip{
		conn: conn,
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		logging.Warning.Printf("Could not create usbip config directory %s: %s", configDir, err)
	}

	err := conn.Export(d, objectPath, ifaceName)
	if err != nil {
		logging.Critical.Panic(err)
	}

	node := &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:    ifaceName,
				Methods: introspect.Methods(d),
			},
		},
	}

	err = conn.Export(introspect.NewIntrospectable(node), objectPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logging.Critical.Panic(err)
	}

	logging.Info.Printf("Exposing object %s with interface %s ...", objectPath, ifaceName)
}
