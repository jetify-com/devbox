package envir

import (
	"os"
	"strconv"

	"go.jetpack.io/devbox/internal/env"
)

func IsCLICloudShell() bool { // TODO: move to env utils
	cliCloudShell, _ := strconv.ParseBool(os.Getenv(env.DevboxCLICloudShell))
	return cliCloudShell
}

func IsDevboxCloud() bool { // TODO: move to env utils
	return GetRegion() != ""
}

func GetRegion() string { // TODO: move to env utils
	return os.Getenv(env.DevboxRegion)
}
