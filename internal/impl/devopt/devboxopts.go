package devopt

import (
	"context"
	"io"
)

type Opts struct {
	Dir            string
	Pure           bool
	IgnoreWarnings bool
	Writer         io.Writer
}

type PrintEnv struct {
	Ctx                  context.Context
	IncludeHooks         bool
	OmitWrappersFromPath bool
}
