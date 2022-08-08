package cgroup

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var deviceCgroupRuleRegex = regexp.MustCompile(`^([acb]) ([0-9]+|\*):([0-9]+|\*) ([rwm]{1,3})$`)

// Lifted from Moby's oci/oci.go
func AppendDevicePermissionsFromCgroupRules(devPermissions []specs.LinuxDeviceCgroup, rules []string) ([]specs.LinuxDeviceCgroup, error) {
	for _, deviceCgroupRule := range rules {
		ss := deviceCgroupRuleRegex.FindAllStringSubmatch(deviceCgroupRule, -1)
		if len(ss) == 0 || len(ss[0]) != 5 {
			return nil, fmt.Errorf("invalid device cgroup rule format: '%s'", deviceCgroupRule)
		}
		matches := ss[0]

		dPermissions := specs.LinuxDeviceCgroup{
			Allow:  true,
			Type:   matches[1],
			Access: matches[4],
		}
		if matches[2] == "*" {
			major := int64(-1)
			dPermissions.Major = &major
		} else {
			major, err := strconv.ParseInt(matches[2], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid major value in device cgroup rule format: '%s'", deviceCgroupRule)
			}
			dPermissions.Major = &major
		}
		if matches[3] == "*" {
			minor := int64(-1)
			dPermissions.Minor = &minor
		} else {
			minor, err := strconv.ParseInt(matches[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid minor value in device cgroup rule format: '%s'", deviceCgroupRule)
			}
			dPermissions.Minor = &minor
		}
		devPermissions = append(devPermissions, dPermissions)
	}
	return devPermissions, nil
}

func CreateDeviceUpdateResources(rules []string) (*specs.LinuxResources, error) {
	resources := specs.LinuxResources{}

	devices, err := AppendDevicePermissionsFromCgroupRules(nil, rules)
	if err != nil {
		return nil, err
	}

	resources.Devices = devices

	return &resources, nil
}
