package envir

import (
	"os"
	"strconv"
)

func IsCLICloudShell() bool {
	cliCloudShell, _ := strconv.ParseBool(os.Getenv("DEVBOX_CLI_CLOUD_SHELL"))
	return cliCloudShell
}

func IsDevboxCloud() bool {
	return GetRegion() != ""
}

func GetRegion() string {
	return os.Getenv("DEVBOX_REGION")
}
