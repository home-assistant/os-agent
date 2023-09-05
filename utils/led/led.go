package led

import (
	"os"
	"path/filepath"
	"strings"
)

type LED struct {
	Name           string
	DefaultTrigger string
}

func (led LED) GetTrigger() (string, error) {
	ledTriggerFilePath := filepath.Join("/sys/class/leds/", led.Name, "trigger")
	ledTrigger, err := os.ReadFile(ledTriggerFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ledTrigger)), err
}

func (led LED) SetTrigger(newState bool) error {
	ledTriggerFilePath := filepath.Join("/sys/class/leds/", led.Name, "trigger")
	var newTrigger []byte

	if newState {
		newTrigger = []byte(led.DefaultTrigger)
	} else {
		newTrigger = []byte("none")
	}

	err := os.WriteFile(ledTriggerFilePath, newTrigger, 0600)
	if err != nil {
		return err
	}

	return nil
}
