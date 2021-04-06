package logging

import (
	"log"
	"os"
)

var (
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	Critical *log.Logger
)

func init() {

	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Critical = log.New(os.Stderr, "CRITICAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}
