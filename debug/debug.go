package debug

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

var enabled bool

func init() {
	enabled, _ = strconv.ParseBool(os.Getenv("DEBUG"))
}

func IsEnabled() bool { return enabled }

func Enable() {
	enabled = true
	log.SetPrefix("[DEBUG] ")
	log.SetFlags(log.Llongfile | log.Ldate | log.Ltime)
	_ = log.Output(2, "Debug mode enabled.")
}

func Log(format string, v ...any) {
	if !enabled {
		return
	}
	_ = log.Output(2, fmt.Sprintf(format, v...))
}

func Recover() {
	r := recover()
	if r == nil {
		return
	}

	if enabled {
		log.Println("Allowing panic because debug mode is enabled.")
		panic(r)
	}
	fmt.Println("Error:", r)
}
