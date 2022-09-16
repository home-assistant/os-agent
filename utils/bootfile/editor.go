package bootfile

import (
	"bufio"
	"os"
	"strings"

	logging "github.com/home-assistant/os-agent/utils/log"
)

func ReadOption(filePath string, optionName string, defaultValue string) (string, error) {
	// Read the options from the boot file
	file, err := os.Open(filePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", filePath, err)
		return defaultValue, err
	}
	defer file.Close()

	// Scan over all lines
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, optionName) {
			return strings.Replace(line, optionName, "", 1), nil
		}
	}

	return defaultValue, nil
}

func DisableOption(filePath string, optionName string) error {
	// Read the options from the boot file
	file, err := os.Open(filePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", filePath, err)
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
	file, err = os.Create(filePath)
	if err != nil {
		logging.Error.Printf("Failed to write boot file %s: %s", filePath, err)
		return err
	}

	writter := bufio.NewWriter(file)
	for _, line := range outLines {
		writter.WriteString(line + "\n")
	}
	writter.Flush()
	file.Close()

	return nil
}

func SetOption(filePath string, optionName string, value string) error {
	// Read the options from the boot file
	file, err := os.Open(filePath)
	if err != nil {
		logging.Error.Printf("Failed to open boot file %s: %s", filePath, err)
		return err
	}

	// Scan over all lines
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var outLines []string
	var found bool = false
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, optionName) || strings.HasPrefix(line, "#"+optionName) {
			if !found {
				outLines = append(outLines, optionName+value)
				found = true
			}
		} else {
			outLines = append(outLines, line)
		}
	}
	file.Close()

	// No option found, add it
	if !found {
		outLines = append(outLines, optionName+value)
	}

	// Write all lines back to boot config file
	file, err = os.Create(filePath)
	if err != nil {
		logging.Error.Printf("Failed to write boot file %s: %s", filePath, err)
		return err
	}

	writter := bufio.NewWriter(file)
	for _, line := range outLines {
		writter.WriteString(line + "\n")
	}
	writter.Flush()
	file.Close()

	return nil
}
