package rpi

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	logging "github.com/home-assistant/os-agent/utils/log"
)

const (
	objectPath = "/io/hass/os/Boards/RaspberryPi/Firmware"
	ifaceName  = "io.hass.os.Boards.RaspberryPi.Firmware"

	eepromUpdateCmd  = "rpi-eeprom-update"
	readStateTimeout = 30 * time.Second
	updateTimeout    = 5 * time.Minute

	blockedReasonBootDevice = "unsupported_boot_device"
	// blockedStatusLine is the line rpi-eeprom-update prints when the current
	// boot device cannot apply an update.
	blockedStatusLine = "BLOCKED: yes"
)

type firmware struct {
	conn  *dbus.Conn
	props *prop.Properties
	state eepromState
}

type eepromState struct {
	currentVersion  string
	latestVersion   string
	updateAvailable bool
	updateBlocked   bool
	blockedReason   string
}

// hasOutputLine reports whether out has a line that, once trimmed, equals want.
func hasOutputLine(out, want string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

var epochRe = regexp.MustCompile(`\((\d+)\)\s*$`)

// extractVersion turns a CURRENT/LATEST value into a stable identifier.
// Bootloader values look like "Wed 14 Apr 2021 ... (1618412973)" - we use the
// epoch number. VL805 values are bare hex like "000138c0" - passed through.
func extractVersion(value string) string {
	value = strings.TrimSpace(value)
	if m := epochRe.FindStringSubmatch(value); m != nil {
		return m[1]
	}
	return value
}

// parseSection returns the first CURRENT and LATEST values found inside the
// named section (e.g. "BOOTLOADER", "VL805") of `rpi-eeprom-update` output.
// Returns empty strings if the section is absent or doesn't contain them.
func parseSection(out, section string) (current, latest string) {
	header := section + ":"
	inSection := false
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if !inSection {
			if strings.HasPrefix(line, header) {
				inSection = true
			}
			continue
		}
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "CURRENT:") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "CURRENT:"))
		} else if strings.HasPrefix(line, "LATEST:") {
			latest = strings.TrimSpace(strings.TrimPrefix(line, "LATEST:"))
		}
		if current != "" && latest != "" {
			return
		}
	}
	return
}

// composeVersions applies the following rule for combining bootloader and VL805
// versions into a single pair of comparable strings:
//   - bootloader update pending → just the bootloader epoch.
//   - bootloader up-to-date, VL805 update pending → "<bootloader>-<vl805>".
//   - everything up-to-date → just the bootloader epoch on both sides.
func composeVersions(blCur, blLat, vlCur, vlLat string) (string, string) {
	if blCur == "" {
		return "", ""
	}
	if blCur != blLat {
		return blCur, blLat
	}
	if vlCur != "" && vlLat != "" && vlCur != vlLat {
		return blCur + "-" + vlCur, blLat + "-" + vlLat
	}
	return blCur, blCur
}

func readState() eepromState {
	ctx, cancel := context.WithTimeout(context.Background(), readStateTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, eepromUpdateCmd)
	out, err := cmd.CombinedOutput()
	outStr := string(out)

	// Exit 0 = up to date, 1 = update required (EXIT_UPDATE_REQUIRED), anything
	// else = failure. Exit 0 also covers an installed firmware newer than the
	// bundled blob, which must not be offered as a (downgrade) update.
	updateRequired := false
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			logging.Error.Printf("rpi-eeprom-update failed: %v: %s", err, outStr)
			return eepromState{}
		}
		if ee.ExitCode() != 1 {
			logging.Error.Printf("rpi-eeprom-update failed (exit %d): %s", ee.ExitCode(), outStr)
			return eepromState{}
		}
		updateRequired = true
	}

	blCur, blLat := parseSection(outStr, "BOOTLOADER")
	vlCur, vlLat := parseSection(outStr, "VL805")
	blCur = extractVersion(blCur)
	blLat = extractVersion(blLat)
	vlCur = extractVersion(vlCur)
	vlLat = extractVersion(vlLat)

	current, latest := composeVersions(blCur, blLat, vlCur, vlLat)
	if !updateRequired {
		// Pin latest to current so an installed firmware newer than the
		// bundled blob isn't surfaced as a downgrade.
		latest = current
	}

	state := eepromState{
		currentVersion:  current,
		latestVersion:   latest,
		updateAvailable: updateRequired,
		updateBlocked:   hasOutputLine(outStr, blockedStatusLine),
	}
	if state.updateBlocked {
		state.blockedReason = blockedReasonBootDevice
	}
	return state
}

// Update applies the bundled EEPROM (and VL805 where present) firmware. The
// new bootloader only takes effect after a reboot, so callers should offer a
// reboot prompt.
func (d firmware) Update() *dbus.Error {
	// Refuse up front so the caller gets a clean error rather than the tool's
	// raw output. Rejecting when no update is available also keeps a no-op run
	// from being surfaced as an applied update needing a reboot.
	if d.state.updateBlocked {
		return dbus.MakeFailedError(fmt.Errorf("EEPROM update is unavailable on this boot device"))
	}
	if !d.state.updateAvailable {
		return dbus.MakeFailedError(fmt.Errorf("no EEPROM update available"))
	}

	logging.Info.Print("Starting EEPROM update via rpi-eeprom-update -a")
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, eepromUpdateCmd, "-a")
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Error.Printf("rpi-eeprom-update -a failed: %v: %s", err, out)
		return dbus.MakeFailedError(fmt.Errorf("rpi-eeprom-update -a failed: %w", err))
	}
	logging.Info.Printf("EEPROM update completed: %s", strings.TrimSpace(string(out)))

	// A reboot is always required to run the new firmware. Hold this in
	// UpdatePending so the state is visible until the device reboots; it
	// resets to false whenever os-agent restarts (i.e. after a reboot).
	d.props.SetMust(ifaceName, "UpdatePending", true)
	return nil
}

func InitializeDBus(conn *dbus.Conn) {
	initial := readState()

	d := firmware{conn: conn, state: initial}

	propsSpec := map[string]map[string]*prop.Prop{
		ifaceName: {
			"CurrentVersion": {
				Value:    initial.currentVersion,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"LatestVersion": {
				Value:    initial.latestVersion,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"UpdateAvailable": {
				Value:    initial.updateAvailable,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"UpdateBlocked": {
				Value:    initial.updateBlocked,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"UpdatePending": {
				Value:    false,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			"BlockedReason": {
				Value:    initial.blockedReason,
				Writable: false,
				Emit:     prop.EmitTrue,
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
