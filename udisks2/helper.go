package udisks2

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
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

func (u UDisks2Helper) GetPartitionDeviceFromLabel(label string) (*string, error) {

	busObject, err := u.manager.ResolveDeviceFromLabel(label)
	if err != nil {
		return nil, err
	}

	busObjectBlock := u.conn.Object("org.freedesktop.UDisks2", *busObject)
	block := NewBlock(busObjectBlock)

	return block.GetDeviceString(context.Background())
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

func (u UDisks2Helper) FormatPartition(devicePath string, fsType string, label string) error {
	devspec := map[string]dbus.Variant{"path": dbus.MakeVariant(devicePath)}
	blockObjects, err := u.manager.ResolveDevice(context.Background(), devspec, noOptions)
	if err != nil {
		return err
	}
	if len(blockObjects) != 1 {
		return fmt.Errorf("Expected single block device with device path \"%s\", found %d", devicePath, len(blockObjects))
	}

	fmt.Printf("Formatting block device %s with file system \"%s\".\n", devicePath, fsType)
	blockObjectPath := blockObjects[0]
	busObjectParentBlock := u.conn.Object("org.freedesktop.UDisks2", blockObjectPath)
	parentBlock := NewBlock(busObjectParentBlock)
	formatOptions := map[string]dbus.Variant{"label": dbus.MakeVariant(label)}
	err = parentBlock.Format(context.Background(), fsType, formatOptions)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully formatted block device %s.\n", devicePath)

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
	fmt.Printf("Formatting device %s\n", devicePath)

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
	fmt.Printf("New partition D-Bus object %s\n", createdPartition)

	return nil
}
