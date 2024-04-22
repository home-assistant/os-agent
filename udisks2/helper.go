package udisks2

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"

	logging "github.com/wit-ds/os-agent/utils/log"
)

type UDisks2Helper struct {
	conn    *dbus.Conn
	manager *Manager
}

func NewUDisks2(conn *dbus.Conn) UDisks2Helper {
	busObj := conn.Object("org.freedesktop.UDisks2", "/org/freedesktop/UDisks2/Manager")
	manager := NewManager(busObj)

	d := UDisks2Helper{
		conn:    conn,
		manager: manager,
	}

	return d
}

func (u UDisks2Helper) GetBusObjectFromLabel(label string) (dbus.BusObject, error) {

	busObject, err := u.manager.ResolveDeviceFromLabel(label)
	if err != nil {
		return nil, err
	}

	return u.conn.Object("org.freedesktop.UDisks2", *busObject), nil
}

func (u UDisks2Helper) GetRootDeviceFromLabel(label string) (*string, error) {

	busObject, err := u.manager.ResolveDeviceFromLabel(label)
	if err != nil {
		return nil, err
	}

	busObjectBlock := u.conn.Object("org.freedesktop.UDisks2", *busObject)
	partition := NewPartition(busObjectBlock)
	table, err := partition.GetTable(context.Background())
	if err != nil {
		return nil, err
	}

	/* Get Block device of partition table */
	busObjectParentBlock := u.conn.Object("org.freedesktop.UDisks2", table)
	parentBlock := NewBlock(busObjectParentBlock)

	return parentBlock.GetDeviceString(context.Background())
}

func (u UDisks2Helper) FormatPartition(blockObjectPath dbus.BusObject, fsType string, label string) error {
	parentBlock := NewBlock(blockObjectPath)
	formatOptions := map[string]dbus.Variant{"label": dbus.MakeVariant(label)}
	err := parentBlock.Format(context.Background(), fsType, formatOptions)
	if err != nil {
		return err
	}

	return nil
}

func (u UDisks2Helper) FormatPartitionFromDevicePath(devicePath string, fsType string, label string) error {
	devspec := map[string]dbus.Variant{"path": dbus.MakeVariant(devicePath)}
	blockObjects, err := u.manager.ResolveDevice(context.Background(), devspec, noOptions)
	if err != nil {
		return err
	}
	if len(blockObjects) != 1 {
		return fmt.Errorf("Expected single block device with device path \"%s\", found %d", devicePath, len(blockObjects))
	}

	logging.Info.Printf("Formatting block device %s with file system \"%s\".", devicePath, fsType)
	blockObjectPath := blockObjects[0]
	busObjectBlock := u.conn.Object("org.freedesktop.UDisks2", blockObjectPath)

	err = u.FormatPartition(busObjectBlock, fsType, label)
	if err != nil {
		return err
	}

	logging.Info.Printf("Successfully formatted block device %s.", devicePath)

	return nil
}

func (u UDisks2Helper) PartitionDeviceWithSinglePartition(devicePath string, uuid string, name string) error {
	devspec := map[string]dbus.Variant{"path": dbus.MakeVariant(devicePath)}
	blockObjects, err := u.manager.ResolveDevice(context.Background(), devspec, noOptions)
	if err != nil {
		return err
	}
	if len(blockObjects) != 1 {
		return fmt.Errorf("Expected single block device with device path \"%s\", found %d", devicePath, len(blockObjects))
	}

	blockObjectPath := blockObjects[0]
	logging.Info.Printf("Formatting device %s", devicePath)

	busObjectParentBlock := u.conn.Object("org.freedesktop.UDisks2", blockObjectPath)
	parentBlock := NewBlock(busObjectParentBlock)
	err = parentBlock.Format(context.Background(), "gpt", noOptions)
	if err != nil {
		return err
	}

	parentPartitionTable := NewPartitionTable(busObjectParentBlock)
	createdPartition, err :=
		parentPartitionTable.CreatePartition(context.Background(), 0, 0,
			uuid, name, noOptions)
	if err != nil {
		return err
	}
	logging.Info.Printf("New partition D-Bus object %s.", createdPartition)

	return nil
}
