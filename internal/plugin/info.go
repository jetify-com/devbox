package plugin

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

func PrintReadme(pkg, projectDir string, w io.Writer, markdown bool) error {
	cfg, err := getConfigIfAny(pkg, projectDir)

	if err != nil {
		return err
	}

	if cfg == nil {
		return nil
	}

	_, _ = fmt.Fprintln(w, "")

	if err = printReadme(cfg, w, markdown); err != nil {
		return err
	}

	if err = printServices(cfg, w, markdown); err != nil {
		return err
	}

	if err = printCreateFiles(cfg, w, markdown); err != nil {
		return err
	}

	if err = printEnv(cfg, w, markdown); err != nil {
		return err
	}

	return printInfoInstructions(pkg, w)
}

func printReadme(cfg *config, w io.Writer, markdown bool) error {
	if cfg.Readme == "" {
		return nil
	}
	_, err := fmt.Fprintf(
		w,
		"%s%s NOTES:\n%s\n\n",
		lo.Ternary(markdown, "### ", ""),
		cfg.Name,
		cfg.Readme,
	)
	return errors.WithStack(err)
}

func printServices(cfg *config, w io.Writer, markdown bool) error {
	if len(cfg.Services) == 0 {
		return nil
	}
	services := ""
	for _, service := range cfg.Services {
		services += fmt.Sprintf("* %[1]s\n", service.Name)
	}

	_, err := fmt.Fprintf(
		w,
		"%sServices:\n%s\nUse `devbox services start|stop [service]` to interact with services\n\n",
		lo.Ternary(markdown, "### ", ""),
		services,
	)
	return errors.WithStack(err)
}

func printCreateFiles(cfg *config, w io.Writer, markdown bool) error {
	if len(cfg.CreateFiles) == 0 {
		return nil
	}

	shims := ""
	for name, src := range cfg.CreateFiles {
		if src != "" {
			shims += fmt.Sprintf("* %s\n", name)
		}
	}

	_, err := fmt.Fprintf(
		w,
		"%sThis plugin creates the following helper files:\n%s\n",
		lo.Ternary(markdown, "### ", ""),
		shims,
	)
	return errors.WithStack(err)
}

func printEnv(cfg *config, w io.Writer, markdown bool) error {
	if len(cfg.Env) == 0 {
		return nil
	}

	envVars := ""
	for name, value := range cfg.Env {
		envVars += fmt.Sprintf("* %s=%s\n", name, value)
	}

	_, err := fmt.Fprintf(
		w,
		"%sThis plugin sets the following environment variables:\n%s\n",
		lo.Ternary(markdown, "### ", ""),
		envVars,
	)
	return errors.WithStack(err)
}

func printInfoInstructions(pkg string, w io.Writer) error {
	_, err := fmt.Fprintf(
		w,
		"To show this information, run `devbox info %s`\n\n",
		pkg,
	)
	return errors.WithStack(err)
}

func PrintEnvUpdateMessage(projectDir string, w io.Writer) {
	msg := "\nTo update your shell and ensure your new packages are usable, " +
		"please run:\n\neval $(devbox shellenv)\n\n\n"
	if refreshAlias := os.Getenv("DEVBOX_REFRESH_ALIAS"); refreshAlias != "" {
		msg = fmt.Sprintf(
			"\nTo update your shell and ensure your new packages are usable, "+
				"please run: %q\n",
			refreshAlias,
		)
	}
	color.New(color.FgYellow).Fprint(w, msg)
}
