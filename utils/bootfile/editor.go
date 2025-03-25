package bootfile

import (
	"bufio"
	"os"
	"strings"

	logging "github.com/home-assistant/os-agent/utils/log"

	"github.com/natefinch/atomic"
)

type Editor struct {
	FilePath  string
	Delimiter string
}

func (e Editor) ReadOption(optionName string, defaultValue string) (string, error) {
	// Read the options from the boot file
	file, err := os.Open(e.FilePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", e.FilePath, err)
		return defaultValue, err
	}
	defer file.Close()

	// Scan over all lines
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, optionName) {
			return strings.Replace(line, optionName+e.Delimiter, "", 1), nil
		}
	}

	return defaultValue, nil
}

func (e Editor) DisableOption(optionName string) error {
	// Read the options from the boot file
	file, err := os.Open(e.FilePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", e.FilePath, err)
		return err
	}

	// Scan over all lines
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var outLines []string
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, optionName) {
			outLines = append(outLines, "#"+line)
		} else {
			outLines = append(outLines, line)
		}
	}
	file.Close()

	// Write all lines back to boot config file
	return e.writeNewBootFile(outLines)
}

func (e Editor) SetOption(optionName string, value string) error {
	// Read the options from the boot file
	file, err := os.Open(e.FilePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", e.FilePath, err)
		return err
	}

	// Scan over all lines
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var outLines []string
	var found = false
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, optionName) || strings.HasPrefix(line, "#"+optionName) {
			if !found {
				outLines = append(outLines, optionName+e.Delimiter+value)
				found = true
			}
		} else {
			outLines = append(outLines, line)
		}
	}
	file.Close()

	// No option found, add it
	if !found {
		outLines = append(outLines, optionName+e.Delimiter+value)
	}

	// Write all lines back to boot config file
	return e.writeNewBootFile(outLines)
}

func (e Editor) writeNewBootFile(lines []string) error {
	// Write all lines back to boot config file
	raw := strings.Join(lines, "\n")
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	reader := strings.NewReader(raw)

	err := atomic.WriteFile(e.FilePath, reader)
	if err != nil {
		logging.Error.Printf("Failed to write boot file %s: %s", e.FilePath, err)
		return err
	}

	return err
}
