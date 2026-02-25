package devbox

import (
	"context"
	"fmt"
	"strconv"
	"text/tabwriter"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/services"
)

func (d *Devbox) StartServices(
	ctx context.Context, runInCurrentShell bool, serviceNames ...string,
) error {
	if !runInCurrentShell {
		return d.runDevboxServicesScript(ctx,
			append(
				[]string{"start", "--run-in-current-shell"},
				serviceNames...,
			),
		)
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.stderr, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.stderr, "\nNOTE: We recommend using `devbox services up` to start process-compose and your services")
		return d.StartProcessManager(ctx, runInCurrentShell, serviceNames, devopt.ProcessComposeOpts{Background: true})
	}

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	if len(svcSet) == 0 {
		return usererr.New("No services found in your project")
	}

	for _, s := range serviceNames {
		if _, ok := svcSet[s]; !ok {
			return usererr.New("Service %s not found in your project", s)
		}
	}

	for _, s := range serviceNames {
		err := services.StartServices(ctx, d.stderr, s, d.projectDir)
		if err != nil {
			fmt.Fprintf(d.stderr, "Error starting service %s: %s", s, err)
		} else {
			fmt.Fprintf(d.stderr, "Service %s started successfully", s)
		}
	}
	return nil
}

func (d *Devbox) StopServices(ctx context.Context, runInCurrentShell, allProjects bool, serviceNames ...string) error {
	if !runInCurrentShell {
		args := []string{"stop", "--run-in-current-shell"}
		args = append(args, serviceNames...)
		if allProjects {
			args = append(args, "--all-projects")
		}
		return d.runDevboxServicesScript(ctx, args)
	}

	if allProjects {
		return services.StopAllProcessManagers(ctx, d.stderr)
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		return usererr.New("Process manager is not running. Run `devbox services up` to start it.")
	}

	if len(serviceNames) == 0 {
		return services.StopProcessManager(ctx, d.projectDir, d.stderr)
	}

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	for _, s := range serviceNames {
		if _, ok := svcSet[s]; !ok {
			return usererr.New("Service %s not found in your project", s)
		}
		err := services.StopServices(ctx, s, d.projectDir, d.stderr)
		if err != nil {
			fmt.Fprintf(d.stderr, "Error stopping service %s: %s", s, err)
		}
	}
	return nil
}

func (d *Devbox) ListServices(ctx context.Context, runInCurrentShell bool) error {
	if !runInCurrentShell {
		return d.runDevboxServicesScript(ctx, []string{"ls", "--run-in-current-shell"})
	}

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	if len(svcSet) == 0 {
		fmt.Fprintln(d.stderr, "No services found in your project")
		return nil
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.stderr, "No services currently running. Run `devbox services up` to start them:")
		fmt.Fprintln(d.stderr, "")
		for _, s := range svcSet {
			fmt.Fprintf(d.stderr, "  %s\n", s.Name)
		}
		return nil
	}
	tw := tabwriter.NewWriter(d.stderr, 3, 2, 8, ' ', tabwriter.TabIndent)
	pcSvcs, err := services.ListServices(ctx, d.projectDir, d.stderr)
	if err != nil {
		fmt.Fprintln(d.stderr, "Error listing services: ", err)
	} else {
		fmt.Fprintln(d.stderr, "Services running in process-compose:")
		fmt.Fprintln(tw, "PID\tNAME\tNAMESPACE\tSTATUS\tAGE\tHEALTH\tRESTARTS\tEXIT CODE")
		for _, s := range pcSvcs {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n", s.PID, s.Name, s.Namespace, s.Status, s.Age, s.Health, s.Restarts, s.ExitCode)
		}
		tw.Flush()
	}
	return nil
}

func (d *Devbox) RestartServices(
	ctx context.Context, runInCurrentShell bool, serviceNames ...string,
) error {
	if !runInCurrentShell {
		return d.runDevboxServicesScript(ctx,
			append(
				[]string{"restart", "--run-in-current-shell"},
				serviceNames...,
			),
		)
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.stderr, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.stderr, "\nTip: We recommend using `devbox services up` to start process-compose and your services")
		return d.StartProcessManager(ctx, runInCurrentShell, serviceNames, devopt.ProcessComposeOpts{Background: true})
	}

	// TODO: Restart with no services should restart the _currently running_ services. This means we should get the list of running services from the process-compose, then restart them all.

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	for _, s := range serviceNames {
		if _, ok := svcSet[s]; !ok {
			return usererr.New("Service %s not found in your project", s)
		}
		err := services.RestartServices(ctx, s, d.projectDir, d.stderr)
		if err != nil {
			fmt.Printf("Error restarting service %s: %s", s, err)
		} else {
			fmt.Printf("Service %s restarted", s)
		}
	}
	return nil
}

func (d *Devbox) AttachToProcessManager(ctx context.Context) error {
	if !services.ProcessManagerIsRunning(d.projectDir) {
		return usererr.New("Process manager is not running. Run `devbox services up` to start it.")
	}

	err := initDevboxUtilityProject(ctx, d.stderr)
	if err != nil {
		return err
	}

	processComposeBinPath, err := utilityLookPath("process-compose")
	if err != nil {
		return err
	}

	return services.AttachToProcessManager(
		ctx,
		d.stderr,
		d.projectDir,
		services.ProcessComposeOpts{
			BinPath: processComposeBinPath,
		},
	)
}

func (d *Devbox) StartProcessManager(
	ctx context.Context,
	runInCurrentShell bool,
	requestedServices []string,
	processComposeOpts devopt.ProcessComposeOpts,
) error {
	if !runInCurrentShell {
		args := []string{"up", "--run-in-current-shell"}
		args = append(args, requestedServices...)

		// TODO: Here we're attempting to reconstruct arguments from the original command, so that we can reinvoke it in devbox shell.
		// 		 Instead, we should consider refactoring this so that we can preserve and re-use the original command string,
		//		 because the current approach is fragile and will need to be updated each time we add new flags.
		if d.customProcessComposeFile != "" {
			args = append(args, "--process-compose-file", d.customProcessComposeFile)
		}
		if processComposeOpts.Background {
			args = append(args, "--background")
		}
		for _, flag := range processComposeOpts.ExtraFlags {
			args = append(args, "--pcflags", flag)
		}
		if processComposeOpts.ProcessComposePort != 0 {
			args = append(args, "--pcport", strconv.Itoa(processComposeOpts.ProcessComposePort))
		}

		return d.runDevboxServicesScript(ctx, args)
	}

	svcs, err := d.Services()
	if err != nil {
		return err
	}

	if len(svcs) == 0 {
		return usererr.New("No services found in your project")
	}

	for _, s := range requestedServices {
		if _, ok := svcs[s]; !ok {
			return usererr.New("Service %s not found in your project", s)
		}
	}

	err = initDevboxUtilityProject(ctx, d.stderr)
	if err != nil {
		return err
	}

	processComposeBinPath, err := utilityLookPath("process-compose")
	if err != nil {
		return err
	}

	// Start the process manager

	return services.StartProcessManager(
		d.stderr,
		requestedServices,
		svcs,
		d.projectDir,
		services.ProcessComposeOpts{
			BinPath:            processComposeBinPath,
			Background:         processComposeOpts.Background,
			ExtraFlags:         processComposeOpts.ExtraFlags,
			ProcessComposePort: processComposeOpts.ProcessComposePort,
		},
	)
}

// runDevboxServicesScript invokes RunScript with the envOptions set to the appropriate
// defaults for the `devbox services` scenario.
func (d *Devbox) runDevboxServicesScript(ctx context.Context, cmdArgs []string) error {
	cmdArgs = append([]string{"services"}, cmdArgs...)
	return d.RunScript(ctx, devopt.EnvOptions{}, "devbox", cmdArgs)
}
