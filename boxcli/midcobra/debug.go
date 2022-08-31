package midcobra

import (
	"log"

	"github.com/spf13/cobra"
)

type Debug bool

var _ Middleware = (*Debug)(nil)

func (d *Debug) preRun(cmd *cobra.Command, args []string) {}

func (d *Debug) postRun(cmd *cobra.Command, args []string, runErr error) {
	if *d {
		log.Printf("Error: %+v\n", runErr)
	}
}
