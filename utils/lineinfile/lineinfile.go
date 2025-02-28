package lineinfile

import (
	"fmt"
	logging "github.com/home-assistant/os-agent/utils/log"
	"github.com/natefinch/atomic"
	"os"
	re "regexp"
	"strings"
)

type LineInFile struct {
	FilePath string
}

type Params struct {
	// Line to insert at the line matching the Regexp, not needed for Absent
	Line string
	// Regular expression matching the line to edit or remove
	Regexp *re.Regexp
	// For Present, insert line after the expression, for Absent, remove line only if it occurs after expression;
	// accepts special "BOF" and "EOF" values (beginning of file, end of file)
	After string
	// For Present, insert line before the expression, for Absent, remove line only if it occurs before expression;
	// accepts special "BOF" and "EOF" values (beginning of file, end of file)
	Before string
}

func NewPresentParams(line string) Params {
	params := Params{
		Line:   line,
		Regexp: nil,
		After:  "EOF",
		Before: "",
	}
	return params
}

func NewAbsentParams() Params {
	params := Params{
		Line:   "",
		Regexp: nil,
		After:  "",
		Before: "EOF",
	}
	return params
}

func (l LineInFile) Present(params Params) error {
	createFile := false
	if _, err := os.Stat(l.FilePath); os.IsNotExist(err) {
		// will be created by atomic.WriteFile
		createFile = true
	} else if err != nil {
		return err
	}

	var lines []string
	if !createFile {
		content, err := os.ReadFile(l.FilePath)
		if err != nil {
			return err
		}

		lines = strings.Split(string(content), "\n")
	}

	outLines, err := processPresent(lines, params)
	if err != nil {
		logging.Error.Printf("Failed to process file %s: %s", l.FilePath, err)
		return err
	}

	err = l.writeFile(outLines)
	if err != nil {
		return err
	}

	return nil
}

func (l LineInFile) Absent(params Params) error {
	if _, err := os.Stat(l.FilePath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	content, err := os.ReadFile(l.FilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	outLines, err := processAbsent(lines, params)
	if err != nil {
		logging.Error.Printf("Failed to process file %s: %s", l.FilePath, err)
		return err
	}

	err = l.writeFile(outLines)
	if err != nil {
		return err
	}

	return nil
}

func processPresent(inLines []string, params Params) ([]string, error) {
	var outLines []string

	if params.Before != "" && params.After != "EOF" {
		err := fmt.Errorf("cannot specify both Before and After")
		return nil, err
	}

	if params.Regexp == nil {
		err := fmt.Errorf("parameter Regexp must be set")
		return nil, err
	}

	needsBefore := params.Before != ""
	needsAfter := params.After != "EOF"
	afterRegexp, _ := re.Compile(params.After)
	beforeRegexp, _ := re.Compile(params.Before)

	var beforeIndex = -1
	var afterIndex = -1
	var foundIndex = -1

	for idx, curr := range inLines {
		outLines = append(outLines, curr)
		if needsBefore && beforeIndex < 0 && beforeRegexp.MatchString(curr) {
			beforeIndex = idx
			continue
		}
		if needsAfter && afterIndex < 0 && afterRegexp.MatchString(curr) {
			afterIndex = idx
			continue
		}
		if ((!needsAfter && !needsBefore) || needsAfter && afterIndex >= 0 || needsBefore && beforeIndex >= 0) && foundIndex < 0 && params.Regexp.MatchString(curr) {
			foundIndex = idx
		}
	}

	if foundIndex >= 0 {
		// replace found line with the params.Line
		outLines[foundIndex] = params.Line
	} else if params.After == "EOF" {
		outLines = append(outLines, params.Line)
	} else if params.Before == "BOF" {
		outLines = append([]string{params.Line}, outLines...)
	} else if params.After != "" {
		if afterIndex >= 0 {
			// insert after the line matching the After regexp
			outLines = append(outLines[:afterIndex+1], append([]string{params.Line}, outLines[afterIndex+1:]...)...)
		}
	}

	return outLines, nil
}

func processAbsent(inLines []string, params Params) ([]string, error) {
	var outLines []string

	if params.Before != "EOF" && params.After != "" {
		err := fmt.Errorf("cannot specify both Before and After")
		return nil, err
	}

	if params.Regexp == nil {
		err := fmt.Errorf("parameter Regexp must be set")
		return nil, err
	}

	needsBefore := params.Before != "EOF"
	needsAfter := params.After != ""
	afterRegexp, _ := re.Compile(params.After)
	beforeRegexp, _ := re.Compile(params.Before)

	var beforeIndex = -1
	var afterIndex = -1
	var foundIndex = -1

	for idx, curr := range inLines {
		outLines = append(outLines, curr)
		if needsBefore && beforeIndex < 0 && beforeRegexp.MatchString(curr) {
			beforeIndex = idx
			continue
		}
		if needsAfter && afterIndex < 0 && afterRegexp.MatchString(curr) {
			afterIndex = idx
			continue
		}
		if ((!needsAfter && !needsBefore) || needsAfter && afterIndex >= 0 || needsBefore && beforeIndex >= 0) && foundIndex < 0 && params.Regexp.MatchString(curr) {
			foundIndex = idx
		}
	}

	if foundIndex >= 0 {
		// remove found line
		outLines = append(outLines[:foundIndex], outLines[foundIndex+1:]...)
	}

	return outLines, nil
}

func (l LineInFile) Find(regexp string, after string, allowMissing bool) (*string, error) {
	if _, err := os.Stat(l.FilePath); os.IsNotExist(err) {
		if allowMissing {
			return nil, nil
		}
		logging.Error.Printf("File %s does not exist: %s", l.FilePath, err)
		return nil, err
	}

	content, err := os.ReadFile(l.FilePath)
	if err != nil {
		logging.Error.Printf("Error reading %s: %s", l.FilePath, err)
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	return processFind(regexp, after, lines), nil
}

func processFind(regexp string, after string, inLines []string) *string {
	if inLines == nil {
		return nil
	}

	lineRegexp, _ := re.Compile(regexp)
	afterRegexp, _ := re.Compile(after)

	var foundAfter = after == ""

	for _, curr := range inLines {
		if !foundAfter && afterRegexp.MatchString(curr) {
			foundAfter = true
			continue
		}
		if after != "" && !foundAfter {
			continue
		}
		if lineRegexp.MatchString(curr) {
			return &curr
		}
	}

	return nil
}

func (l LineInFile) writeFile(lines []string) error {
	raw := strings.Join(lines, "\n")
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	reader := strings.NewReader(raw)

	err := atomic.WriteFile(l.FilePath, reader)

	if err != nil {
		logging.Error.Printf("Failed to write file %s: %s", l.FilePath, err)
		return err
	}

	return nil
}
