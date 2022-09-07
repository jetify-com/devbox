package debug

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

var enabled bool

func init() {
	enabled, _ = strconv.ParseBool(os.Getenv("DEVBOX_DEBUG"))
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

func EarliestStackTrace(err error) errors.StackTrace {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	type causer interface {
		Cause() error
	}

	var st stackTracer
	var earliestStackTrace errors.StackTrace

	for err != nil {
		if errors.As(err, &st) {
			earliestStackTrace = st.StackTrace()
		}

		var c causer
		if !errors.As(err, &c) {
			break
		}
		err = c.Cause()
	}

	return earliestStackTrace
}
