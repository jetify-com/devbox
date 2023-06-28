// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package impl creates isolated development environments.
package impl

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/impl/generate"
	"go.jetpack.io/devbox/internal/shellgen"
	"go.jetpack.io/devbox/internal/telemetry"
	"golang.org/x/exp/slices"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/conf"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/impl/devopt"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/services"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

const (

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"

	arbitraryCmdFilename = ".cmd"
)

type Devbox struct {
	cfg           *devconfig.Config
	lockfile      *lock.File
	nix           nix.Nixer
	projectDir    string
	pluginManager *plugin.Manager
	pure          bool

	// Possible TODO: hardcode this to stderr. Allowing the caller to specify the
	// writer is error prone. Since it is almost always stderr, we should default
	// it and if the user wants stdout then they can return a string and print it.
	// I can't think of a case where we want all the diagnostics to go to stdout.
	writer io.Writer
}

var legacyPackagesWarningHasBeenShown = false

func Open(opts *devopt.Opts) (*Devbox, error) {
	projectDir, err := findProjectDir(opts.Dir)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(projectDir, devconfig.DefaultName)

	cfg, err := devconfig.Load(cfgPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	box := &Devbox{
		cfg:           cfg,
		nix:           &nix.Nix{},
		projectDir:    projectDir,
		pluginManager: plugin.NewManager(),
		writer:        opts.Writer,
		pure:          opts.Pure,
	}

	// TODO savil: this is bad for perf, and so remove before enabling feature.
	// this hack is to workaround an import cycle: lock -> nix -> lock
	userSystem := ""
	if featureflag.RemoveNixpkgs.Enabled() {
		userSystem, err = nix.System()
		if err != nil {
			return nil, err
		}
	}
	lock, err := lock.GetFile(box, searcher.Client(), userSystem)
	if err != nil {
		return nil, err
	}
	box.pluginManager.ApplyOptions(
		plugin.WithDevbox(box),
		plugin.WithLockfile(lock),
	)
	box.lockfile = lock

	if !opts.IgnoreWarnings &&
		!legacyPackagesWarningHasBeenShown &&
		box.HasDeprecatedPackages() {
		legacyPackagesWarningHasBeenShown = true
		globalPath, err := GlobalDataPath()
		if err != nil {
			return nil, err
		}
		ux.Fwarning(
			os.Stderr, // Always stderr. box.writer should probably always be err.
			"Your devbox.json contains packages in legacy format. "+
				"Please run `devbox %supdate` to update your devbox.json.\n",
			lo.Ternary(box.projectDir == globalPath, "global ", ""),
		)
	}

	return box, nil
}

func (d *Devbox) ProjectDir() string {
	return d.projectDir
}

func (d *Devbox) Config() *devconfig.Config {
	return d.cfg
}

func (d *Devbox) ConfigHash() (string, error) {
	hashes := lo.Map(d.PackagesAsInputs(), func(i *nix.Package, _ int) string { return i.Hash() })
	h, err := d.cfg.Hash()
	if err != nil {
		return "", err
	}
	return cuecfg.Hash(h + strings.Join(hashes, ""))
}

func (d *Devbox) NixPkgsCommitHash() string {
	return d.cfg.NixPkgsCommitHash()
}

func (d *Devbox) Generate(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerate")
	defer task.End()

	return errors.WithStack(shellgen.GenerateForPrintEnv(ctx, d))
}

func (d *Devbox) Shell(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "devboxShell")
	defer task.End()

	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}
	fmt.Fprintln(d.writer, "Starting a devbox shell...")

	profileDir, err := d.profilePath()
	if err != nil {
		return err
	}

	envs, err := d.nixEnv(ctx)
	if err != nil {
		return err
	}
	// Used to determine whether we're inside a shell (e.g. to prevent shell inception)
	envs[envir.DevboxShellEnabled] = "1"

	if err := wrapnix.CreateWrappers(ctx, d); err != nil {
		return err
	}

	if err = createDevboxSymlink(d); err != nil {
		return err
	}

	opts := []ShellOption{
		WithHooksFilePath(shellgen.ScriptPath(d.ProjectDir(), shellgen.HooksFilename)),
		WithProfile(profileDir),
		WithHistoryFile(filepath.Join(d.projectDir, shellHistoryFile)),
		WithProjectDir(d.projectDir),
		WithEnvVariables(envs),
		WithShellStartTime(telemetry.ShellStart()),
	}

	shell, err := NewDevboxShell(d, opts...)
	if err != nil {
		return err
	}

	return shell.Run()
}

func (d *Devbox) RunScript(ctx context.Context, cmdName string, cmdArgs []string) error {
	ctx, task := trace.NewTask(ctx, "devboxRun")
	defer task.End()

	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	if err := shellgen.WriteScriptsToFiles(d); err != nil {
		return err
	}

	env, err := d.nixEnv(ctx)
	if err != nil {
		return err
	}
	// Used to determine whether we're inside a shell (e.g. to prevent shell inception)
	// This is temporary because StartServices() needs it but should be replaced with
	// better alternative since devbox run and devbox shell are not the same.
	env["DEVBOX_SHELL_ENABLED"] = "1"

	if err = wrapnix.CreateWrappers(ctx, d); err != nil {
		return err
	}

	// wrap the arg in double-quotes, and escape any double-quotes inside it
	for idx, arg := range cmdArgs {
		cmdArgs[idx] = strconv.Quote(arg)
	}

	var cmdWithArgs []string
	if _, ok := d.cfg.Scripts()[cmdName]; ok {
		// it's a script, so replace the command with the script file's path.
		cmdWithArgs = append([]string{shellgen.ScriptPath(d.ProjectDir(), cmdName)}, cmdArgs...)
	} else {
		// Arbitrary commands should also run the hooks, so we write them to a file as well. However, if the
		// command args include env variable evaluations, then they'll be evaluated _before_ the hooks run,
		// which we don't want. So, one solution is to write the entire command and its arguments into the
		// file itself, but that may not be great if the variables contain sensitive information. Instead,
		// we save the entire command (with args) into the DEVBOX_RUN_CMD var, and then the script evals it.
		err := shellgen.WriteScriptFile(d, arbitraryCmdFilename, shellgen.ScriptBody(d, "eval $DEVBOX_RUN_CMD\n"))
		if err != nil {
			return err
		}
		cmdWithArgs = []string{shellgen.ScriptPath(d.ProjectDir(), arbitraryCmdFilename)}
		env["DEVBOX_RUN_CMD"] = strings.Join(append([]string{cmdName}, cmdArgs...), " ")
	}

	return nix.RunScript(d.projectDir, strings.Join(cmdWithArgs, " "), env)
}

// Install ensures that all the packages in the config are installed and
// creates all wrappers, but does not run init hooks. It is used to power
// devbox install cli command.
func (d *Devbox) Install(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "devboxInstall")
	defer task.End()

	if _, err := d.PrintEnv(ctx, false /*includeHooks*/); err != nil {
		return err
	}
	return wrapnix.CreateWrappers(ctx, d)
}

func (d *Devbox) ListScripts() []string {
	scripts := d.cfg.Scripts()
	keys := make([]string, len(scripts))
	i := 0
	for k := range scripts {
		keys[i] = k
		i++
	}
	return keys
}

func (d *Devbox) PrintEnv(ctx context.Context, includeHooks bool) (string, error) {
	ctx, task := trace.NewTask(ctx, "devboxPrintEnv")
	defer task.End()

	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return "", err
	}

	envs, err := d.nixEnv(ctx)
	if err != nil {
		return "", err
	}

	envStr := exportify(envs)

	if includeHooks {
		hooksStr := ". " + shellgen.ScriptPath(d.ProjectDir(), shellgen.HooksFilename)
		envStr = fmt.Sprintf("%s\n%s;\n", envStr, hooksStr)
	}

	return envStr, nil
}

func (d *Devbox) PrintEnvVars(ctx context.Context) ([]string, error) {
	ctx, task := trace.NewTask(ctx, "devboxPrintEnvVars")
	defer task.End()
	// this only returns env variables for the shell environment excluding hooks
	// and excluding "export " prefix in "export key=value" format
	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return nil, err
	}

	envs, err := d.nixEnv(ctx)
	if err != nil {
		return nil, err
	}
	return keyEqualsValue(envs), nil
}

func (d *Devbox) ShellEnvHash(ctx context.Context) (string, error) {
	envs, err := d.nixEnv(ctx)
	if err != nil {
		return "", err
	}

	return envs[d.ShellEnvHashKey()], nil
}

func (d *Devbox) ShellEnvHashKey() string {
	// Don't make this a const so we don't use it by itself accidentally
	return "__DEVBOX_SHELLENV_HASH_" + d.projectDirHash()
}

func (d *Devbox) Info(ctx context.Context, pkg string, markdown bool) error {
	ctx, task := trace.NewTask(ctx, "devboxInfo")
	defer task.End()

	info := nix.PkgInfo(pkg, d.lockfile)
	if info == nil {
		_, err := fmt.Fprintf(d.writer, "Package %s not found\n", pkg)
		return errors.WithStack(err)
	}
	if _, err := fmt.Fprintf(
		d.writer,
		"%s%s\n",
		lo.Ternary(markdown, "## ", ""),
		info,
	); err != nil {
		return errors.WithStack(err)
	}
	return plugin.PrintReadme(
		ctx,
		nix.PackageFromString(pkg, d.lockfile),
		d.projectDir,
		d.writer,
		markdown,
	)
}

// GenerateDevcontainer generates devcontainer.json and Dockerfile for vscode run-in-container
// and GitHub Codespaces
func (d *Devbox) GenerateDevcontainer(ctx context.Context, force bool) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerateDevcontainer")
	defer task.End()

	// construct path to devcontainer directory
	devContainerPath := filepath.Join(d.projectDir, ".devcontainer/")
	devContainerJSONPath := filepath.Join(devContainerPath, "devcontainer.json")
	dockerfilePath := filepath.Join(devContainerPath, "Dockerfile")

	// check if devcontainer.json or Dockerfile exist
	filesExist := fileutil.Exists(devContainerJSONPath) || fileutil.Exists(dockerfilePath)
	if !force && filesExist {
		return usererr.New(
			"Files devcontainer.json or Dockerfile are already present in .devcontainer/. " +
				"Remove the files or use --force to overwrite them.",
		)
	}

	// create directory
	err := os.MkdirAll(devContainerPath, os.ModePerm)
	if err != nil {
		return redact.Errorf("error creating dev container directory in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	// generate dockerfile
	err = generate.CreateDockerfile(ctx,
		devContainerPath, d.getLocalFlakesDirs(), true /* isDevcontainer */)
	if err != nil {
		return redact.Errorf("error generating dev container Dockerfile in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	// generate devcontainer.json
	err = generate.CreateDevcontainer(ctx, devContainerPath, d.Packages())
	if err != nil {
		return redact.Errorf("error generating devcontainer.json in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	return nil
}

// GenerateDockerfile generates a Dockerfile that replicates the devbox shell
func (d *Devbox) GenerateDockerfile(ctx context.Context, force bool) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerateDockerfile")
	defer task.End()

	dockerfilePath := filepath.Join(d.projectDir, "Dockerfile")
	// check if Dockerfile doesn't exist
	filesExist := fileutil.Exists(dockerfilePath)
	if !force && filesExist {
		return usererr.New(
			"Dockerfile is already present in the current directory. " +
				"Remove it or use --force to overwrite it.",
		)
	}

	// generate dockerfile
	return errors.WithStack(
		generate.CreateDockerfile(ctx,
			d.projectDir, d.getLocalFlakesDirs(), false /* isDevcontainer */))
}

func PrintEnvrcContent(w io.Writer) error {
	return generate.EnvrcContent(w)
}

// GenerateEnvrcFile generates a .envrc file that makes direnv integration convenient
func (d *Devbox) GenerateEnvrcFile(ctx context.Context, force bool) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerateEnvrc")
	defer task.End()

	envrcfilePath := filepath.Join(d.projectDir, ".envrc")
	filesExist := fileutil.Exists(envrcfilePath)
	if !force && filesExist {
		return usererr.New(
			"A .envrc is already present in the current directory. " +
				"Remove it or use --force to overwrite it.",
		)
	}
	// confirm .envrc doesn't exist and don't overwrite an existing .envrc
	if err := nix.EnsureNixInstalled(
		d.writer, func() *bool { return lo.ToPtr(false) },
	); err != nil {
		return err
	}

	// generate all shell files to ensure we can refer to them in the .envrc script
	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	// .envrc file creation
	err := generate.CreateEnvrc(ctx, d.projectDir)
	if err != nil {
		return errors.WithStack(err)
	}
	ux.Fsuccess(d.writer, "generated .envrc file\n")
	if cmdutil.Exists("direnv") {
		cmd := exec.Command("direnv", "allow")
		err := cmd.Run()
		if err != nil {
			return errors.WithStack(err)
		}
		ux.Fsuccess(d.writer, "ran `direnv allow`\n")
	}
	return nil
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	return d.cfg.SaveTo(d.ProjectDir())
}

func (d *Devbox) Services() (services.Services, error) {
	pluginSvcs, err := d.pluginManager.GetServices(
		d.PackagesAsInputs(),
		d.cfg.Include,
	)
	if err != nil {
		return nil, err
	}

	userSvcs := services.FromUserProcessCompose(d.projectDir)

	svcSet := lo.Assign(pluginSvcs, userSvcs)
	keys := make([]string, 0, len(svcSet))
	for k := range svcSet {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := services.Services{}
	for _, k := range keys {
		result[k] = svcSet[k]
	}

	return result, nil
}

func (d *Devbox) StartServices(ctx context.Context, serviceNames ...string) error {
	if !d.IsEnvEnabled() {
		return d.RunScript(ctx, "devbox", append([]string{"services", "start"}, serviceNames...))
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.writer, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.writer, "\nNOTE: We recommend using `devbox services up` to start process-compose and your services")
		return d.StartProcessManager(ctx, serviceNames, true, "")
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
			return usererr.New(fmt.Sprintf("Service %s not found in your project", s))
		}
	}

	for _, s := range serviceNames {
		err := services.StartServices(ctx, d.writer, s, d.projectDir)
		if err != nil {
			fmt.Fprintf(d.writer, "Error starting service %s: %s", s, err)
		} else {
			fmt.Fprintf(d.writer, "Service %s started successfully", s)
		}
	}
	return nil
}

func (d *Devbox) StopServices(ctx context.Context, allProjects bool, serviceNames ...string) error {
	if !d.IsEnvEnabled() {
		args := []string{"services", "stop"}
		args = append(args, serviceNames...)
		if allProjects {
			args = append(args, "--all-projects")
		}
		return d.RunScript(ctx, "devbox", args)
	}

	if allProjects {
		return services.StopAllProcessManagers(ctx, d.writer)
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		return usererr.New("Process manager is not running. Run `devbox services up` to start it.")
	}

	if len(serviceNames) == 0 {
		return services.StopProcessManager(ctx, d.projectDir, d.writer)
	}

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	for _, s := range serviceNames {
		if _, ok := svcSet[s]; !ok {
			return usererr.New(fmt.Sprintf("Service %s not found in your project", s))
		}
		err := services.StopServices(ctx, s, d.projectDir, d.writer)
		if err != nil {
			fmt.Fprintf(d.writer, "Error stopping service %s: %s", s, err)
		}
	}
	return nil
}

func (d *Devbox) ListServices(ctx context.Context) error {
	if !d.IsEnvEnabled() {
		return d.RunScript(ctx, "devbox", []string{"services", "ls"})
	}

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	if len(svcSet) == 0 {
		fmt.Fprintln(d.writer, "No services found in your project")
		return nil
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.writer, "No services currently running. Run `devbox services up` to start them:")
		fmt.Fprintln(d.writer, "")
		for _, s := range svcSet {
			fmt.Fprintf(d.writer, "  %s\n", s.Name)
		}
		return nil
	}
	tw := tabwriter.NewWriter(d.writer, 3, 2, 8, ' ', tabwriter.TabIndent)
	pcSvcs, err := services.ListServices(ctx, d.projectDir, d.writer)
	if err != nil {
		fmt.Fprintln(d.writer, "Error listing services: ", err)
	} else {
		fmt.Fprintln(d.writer, "Services running in process-compose:")
		fmt.Fprintln(tw, "NAME\tSTATUS\tEXIT CODE")
		for _, s := range pcSvcs {
			fmt.Fprintf(tw, "%s\t%s\t%d\n", s.Name, s.Status, s.ExitCode)
		}
		tw.Flush()
	}
	return nil
}

func (d *Devbox) RestartServices(ctx context.Context, serviceNames ...string) error {
	if !d.IsEnvEnabled() {
		return d.RunScript(ctx, "devbox", append([]string{"services", "restart"}, serviceNames...))
	}

	if !services.ProcessManagerIsRunning(d.projectDir) {
		fmt.Fprintln(d.writer, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.writer, "\nTip: We recommend using `devbox services up` to start process-compose and your services")
		return d.StartProcessManager(ctx, serviceNames, true, "")
	}

	// TODO: Restart with no services should restart the _currently running_ services. This means we should get the list of running services from the process-compose, then restart them all.

	svcSet, err := d.Services()
	if err != nil {
		return err
	}

	for _, s := range serviceNames {
		if _, ok := svcSet[s]; !ok {
			return usererr.New(fmt.Sprintf("Service %s not found in your project", s))
		}
		err := services.RestartServices(ctx, s, d.projectDir, d.writer)
		if err != nil {
			fmt.Printf("Error restarting service %s: %s", s, err)
		} else {
			fmt.Printf("Service %s restarted", s)
		}
	}
	return nil
}

func (d *Devbox) StartProcessManager(
	ctx context.Context,
	requestedServices []string,
	background bool,
	processComposeFileOrDir string,
) error {
	svcs, err := d.Services()
	if err != nil {
		return err
	}

	if len(svcs) == 0 {
		return usererr.New("No services found in your project")
	}

	for _, s := range requestedServices {
		if _, ok := svcs[s]; !ok {
			return usererr.New(fmt.Sprintf("Service %s not found in your project", s))
		}
	}

	processComposePath, err := utilityLookPath("process-compose")
	if err != nil {
		fmt.Fprintln(d.writer, "Installing process-compose. This may take a minute but will only happen once.")
		if err = d.addDevboxUtilityPackage("github:F1bonacc1/process-compose/v0.43.1"); err != nil {
			return err
		}

		// re-lookup the path to process-compose
		processComposePath, err = utilityLookPath("process-compose")
		if err != nil {
			fmt.Fprintln(d.writer, "failed to find process-compose after installing it.")
			return err
		}
	}
	if !d.IsEnvEnabled() {
		args := []string{"services", "up"}
		args = append(args, requestedServices...)
		if processComposeFileOrDir != "" {
			args = append(args, "--process-compose-file", processComposeFileOrDir)
		}
		if background {
			args = append(args, "--background")
		}
		return d.RunScript(ctx, "devbox", args)
	}

	// Start the process manager

	return services.StartProcessManager(
		ctx,
		d.writer,
		requestedServices,
		svcs,
		d.projectDir,
		processComposePath, processComposeFileOrDir,
		background,
	)
}

// computeNixEnv computes the set of environment variables that define a Devbox
// environment. The "devbox run" and "devbox shell" commands source these
// variables into a shell before executing a command or showing an interactive
// prompt.
//
// The process for building the environment involves layering sets of
// environment variables on top of each other, with each layer overwriting any
// duplicate keys from the previous:
//
//  1. Copy variables from the current environment except for those in
//     ignoreCurrentEnvVar, such as PWD and SHELL.
//  2. Copy variables from "nix print-dev-env" except for those in
//     ignoreDevEnvVar, such as TMPDIR and HOME.
//  3. Copy variables from Devbox plugins.
//  4. Set PATH to the concatenation of the PATHs from step 3, step 2, and
//     step 1 (in that order).
//
// The final result is a set of environment variables where Devbox plugins have
// the highest priority, then Nix environment variables, and then variables
// from the current environment. Similarly, the PATH gives Devbox plugin
// binaries the highest priority, then Nix packages, and then non-Nix
// programs.
//
// Note that the shellrc.tmpl template (which sources this environment) does
// some additional processing. The computeNixEnv environment won't necessarily
// represent the final "devbox run" or "devbox shell" environments.
func (d *Devbox) computeNixEnv(ctx context.Context, usePrintDevEnvCache bool) (map[string]string, error) {
	defer trace.StartRegion(ctx, "computeNixEnv").End()

	// Append variables from current env if --pure is not passed
	currentEnv := os.Environ()
	env, err := d.parseEnvAndExcludeSpecialCases(currentEnv)
	if err != nil {
		return nil, err
	}

	// check if contents of .envrc is old and print warning
	if !usePrintDevEnvCache {
		err := d.checkOldEnvrc()
		if err != nil {
			return nil, err
		}
	}

	currentEnvPath := env["PATH"]
	debug.Log("current environment PATH is: %s", currentEnvPath)
	// Use the original path, if available. If not available, set it for future calls.
	// See https://github.com/jetpack-io/devbox/issues/687
	// We add the project dir hash to ensure that we don't have conflicts
	// between different projects (including global)
	// (moving a project would change the hash and that's fine)
	originalPath, ok := env[d.ogPathKey()]
	if !ok {
		env[d.ogPathKey()] = currentEnvPath
		originalPath = currentEnvPath
	}

	vaf, err := d.nix.PrintDevEnv(ctx, &nix.PrintDevEnvArgs{
		FlakesFilePath:       d.nixFlakesFilePath(),
		PrintDevEnvCachePath: d.nixPrintDevEnvCachePath(),
		UsePrintDevEnvCache:  usePrintDevEnvCache,
	})
	if err != nil {
		return nil, err
	}

	// Add environment variables from "nix print-dev-env" except for a few
	// special ones we need to ignore.
	for key, val := range vaf.Variables {
		// We only care about "exported" because the var and array types seem to only be used by nix-defined
		// functions that we don't need (like genericBuild). For reference, each type translates to bash as follows:
		// var: export VAR=VAL
		// exported: export VAR=VAL
		// array: declare -a VAR=('VAL1' 'VAL2' )
		if val.Type != "exported" {
			continue
		}

		// SSL_CERT_FILE is a special-case. We only ignore it if it's
		// set to a specific value. This emulates the behavior of
		// "nix develop".
		if key == "SSL_CERT_FILE" && val.Value.(string) == "/no-cert-file.crt" {
			continue
		}

		// Certain variables get set to invalid values after Nix builds
		// the shell environment. For example, HOME=/homeless-shelter
		// and TMPDIR points to a missing directory. We want to ignore
		// those values and just use the values from the current
		// environment instead.
		if ignoreDevEnvVar[key] {
			continue
		}

		env[key] = val.Value.(string)
	}

	// These variables are only needed for shell, but we include them here in the computed env
	// for both shell and run in order to be as identical as possible.
	env["__ETC_PROFILE_NIX_SOURCED"] = "1" // Prevent user init file from loading nix profiles

	debug.Log("nix environment PATH is: %s", env)

	// Add any vars defined in plugins.
	// TODO: Now that we have bin wrappers, this may can eventually be removed.
	// We still need to be able to add env variables to non-service binaries
	// (e.g. ruby). This would involve understanding what binaries are associated
	// to a given plugin.
	pluginEnv, err := d.pluginManager.Env(d.PackagesAsInputs(), d.cfg.Include, env)
	if err != nil {
		return nil, err
	}

	addEnvIfNotPreviouslySetByDevbox(env, pluginEnv)

	envPaths := []string{}
	if !featureflag.PromptHook.Enabled() {
		envPaths = append(envPaths, filepath.Join(d.projectDir, plugin.WrapperBinPath))
	}
	// Adding profile bin path is a temporary hack. Some packages .e.g. curl
	// don't export the correct bin in the package, instead they export
	// as a propagated build input. This can be fixed in 2 ways:
	// * have NixBins() recursively look for bins in propagated build inputs
	// * Turn existing planners into flakes (i.e. php, haskell) and use the bins
	// in the profile.
	// Landau: I prefer option 2 because it doesn't require us to re-implement
	// nix recursive bin lookup.
	envPaths = append(envPaths, nix.ProfileBinPath(d.projectDir), env["PATH"])

	// Prepend virtenv bin path first so user can override it if needed. Virtenv
	// is where the bin wrappers live
	env["PATH"] = JoinPathLists(envPaths...)

	// Include env variables in devbox.json
	configEnv := d.configEnvs(env)
	addEnvIfNotPreviouslySetByDevbox(env, configEnv)

	markEnvsAsSetByDevbox(pluginEnv, configEnv)

	nixEnvPath := env["PATH"]
	debug.Log("PATH after plugins and config is: %s", nixEnvPath)

	// We filter out nix store paths of devbox-packages (represented here as buildInputs).
	// Motivation: if a user removes a package from their devbox it should no longer
	// be available in their environment.
	buildInputs := strings.Split(env["buildInputs"], " ")
	nixEnvPath = filterPathList(nixEnvPath, func(path string) bool {
		for _, input := range buildInputs {
			// input is of the form: /nix/store/<hash>-<package-name>-<version>
			// path is of the form: /nix/store/<hash>-<package-name>-<version>/bin
			if strings.TrimSpace(input) != "" && strings.HasPrefix(path, input) {
				debug.Log("returning false for path %s and input %s\n", path, input)
				return false
			}
		}
		return true
	})
	debug.Log("PATH after filtering with buildInputs (%v) is: %s", buildInputs, nixEnvPath)

	env["PATH"] = JoinPathLists(nixEnvPath, originalPath)
	debug.Log("computed environment PATH is: %s", env["PATH"])

	d.setCommonHelperEnvVars(env)

	if !d.pure {
		// preserve the original XDG_DATA_DIRS by prepending to it
		env["XDG_DATA_DIRS"] = JoinPathLists(
			env["XDG_DATA_DIRS"],
			os.Getenv("XDG_DATA_DIRS"),
		)
	}

	return env, d.addHashToEnv(env)
}

var nixEnvCache map[string]string

// nixEnv is a wrapper around computeNixEnv that caches the result.
// Note that this is in-memory cache of the final environment, and not the same
// as the nix print-dev-env cache which is stored in a file.
func (d *Devbox) nixEnv(ctx context.Context) (map[string]string, error) {
	if nixEnvCache != nil {
		return nixEnvCache, nil
	}

	usePrintDevEnvCache := false

	// If lockfile is up-to-date, we can use the print-dev-env cache.
	lockFile, err := lock.Local(d)
	if err != nil {
		return nil, err
	}
	upToDate, err := lockFile.IsUpToDate()
	if err != nil {
		return nil, err
	}
	if upToDate {
		usePrintDevEnvCache = true
	}

	return d.computeNixEnv(ctx, usePrintDevEnvCache)
}

func (d *Devbox) ogPathKey() string {
	return "DEVBOX_OG_PATH_" + d.projectDirHash()
}

func (d *Devbox) nixPrintDevEnvCachePath() string {
	return filepath.Join(d.projectDir, ".devbox/.nix-print-dev-env-cache")
}

func (d *Devbox) nixFlakesFilePath() string {
	return filepath.Join(d.projectDir, ".devbox/gen/flake/flake.nix")
}

// Packages returns the list of Packages to be installed in the nix shell.
func (d *Devbox) Packages() []string {
	return d.cfg.Packages
}

func (d *Devbox) PackagesAsInputs() []*nix.Package {
	return nix.PackageFromStrings(d.Packages(), d.lockfile)
}

func (d *Devbox) HasDeprecatedPackages() bool {
	for _, pkg := range d.PackagesAsInputs() {
		if pkg.IsLegacy() {
			return true
		}
	}
	return false
}

func (d *Devbox) findPackageByName(name string) (string, error) {
	results := map[string]bool{}
	for _, pkg := range d.cfg.Packages {
		i := nix.PackageFromString(pkg, d.lockfile)
		if i.String() == name || i.CanonicalName() == name {
			results[i.String()] = true
		}
	}
	if len(results) > 1 {
		return "", usererr.New(
			"found multiple packages with name %s: %s. Please specify version",
			name,
			lo.Keys(results),
		)
	}
	if len(results) == 0 {
		return "", usererr.New("no package found with name %s", name)
	}
	return lo.Keys(results)[0], nil
}

func (d *Devbox) checkOldEnvrc() error {
	envrcPath := filepath.Join(d.ProjectDir(), ".envrc")
	noUpdate, err := strconv.ParseBool(os.Getenv("DEVBOX_NO_ENVRC_UPDATE"))
	if err != nil {
		// DEVBOX_NO_ENVRC_UPDATE is either not set or invalid
		// so we consider it the same as false
		noUpdate = false
	}
	// check if user has an old version of envrc
	if fileutil.Exists(envrcPath) && !noUpdate {
		isNewEnvrc, err := fileutil.FileContains(
			envrcPath,
			"eval \"$(devbox generate direnv --print-envrc)\"",
		)
		if err != nil {
			return err
		}
		if !isNewEnvrc {
			ux.Fwarning(
				d.writer,
				"Your .envrc file seems to be out of date. "+
					"Run `devbox generate direnv --force` to update it.\n"+
					"Or silence this warning by setting DEVBOX_NO_ENVRC_UPDATE=1 env variable.\n",
			)
		}
	}
	return nil
}

// configEnvs takes the computed env variables (nix + plugin) and adds env
// variables defined in Config. It also parses variables in config
// that are referenced by $VAR or ${VAR} and replaces them with
// their value in the computed env variables. Note, this doesn't
// allow env variables from outside the shell to be referenced so
// no leaked variables are caused by this function.
func (d *Devbox) configEnvs(computedEnv map[string]string) map[string]string {
	return conf.OSExpandEnvMap(d.cfg.Env, computedEnv, d.ProjectDir())
}

// ignoreCurrentEnvVar contains environment variables that Devbox should remove
// from the slice of [os.Environ] variables before sourcing them. These are
// variables that are set automatically by a new shell.
var ignoreCurrentEnvVar = map[string]bool{
	// Devbox may change the working directory of the shell, so using the
	// original PWD and OLDPWD would be wrong.
	"PWD":    true,
	"OLDPWD": true,

	// SHLVL is the number of nested shells. Copying it would give the
	// Devbox shell the same level as the parent shell.
	"SHLVL": true,

	// The parent shell isn't guaranteed to be the same as the Devbox shell.
	"SHELL": true,

	// The "_" variable is read-only, so we ignore it to avoid attempting to write it later.
	"_": true,
}

// ignoreDevEnvVar contains environment variables that Devbox should remove from
// the slice of [Devbox.PrintDevEnv] variables before sourcing them.
//
// This list comes directly from the "nix develop" source:
// https://github.com/NixOS/nix/blob/f08ad5bdbac02167f7d9f5e7f9bab57cf1c5f8c4/src/nix/develop.cc#L257-L275
var ignoreDevEnvVar = map[string]bool{
	"BASHOPTS":           true,
	"HOME":               true,
	"NIX_BUILD_TOP":      true,
	"NIX_ENFORCE_PURITY": true,
	"NIX_LOG_FD":         true,
	"NIX_REMOTE":         true,
	"PPID":               true,
	"SHELL":              true,
	"SHELLOPTS":          true,
	"TEMP":               true,
	"TEMPDIR":            true,
	"TERM":               true,
	"TMP":                true,
	"TMPDIR":             true,
	"TZ":                 true,
	"UID":                true,
}

// setCommonHelperEnvVars sets environment variables that are required by some
// common setups (e.g. gradio, rust)
func (d *Devbox) setCommonHelperEnvVars(env map[string]string) {
	env["LD_LIBRARY_PATH"] = filepath.Join(d.projectDir, nix.ProfilePath, "lib") + ":" + env["LD_LIBRARY_PATH"]
	env["LIBRARY_PATH"] = filepath.Join(d.projectDir, nix.ProfilePath, "lib") + ":" + env["LIBRARY_PATH"]
}

// NixBins returns the paths to all the nix binaries that are installed by
// the flake. If there are conflicts, it returns the first one it finds of a
// give name. This matches how nix flakes behaves if there are conflicts in
// buildInputs
func (d *Devbox) NixBins(ctx context.Context) ([]string, error) {
	env, err := d.nixEnv(ctx)
	if err != nil {
		return nil, err
	}
	dirs := strings.Split(env["buildInputs"], " ")
	bins := map[string]string{}
	for _, dir := range dirs {
		binPath := filepath.Join(dir, "bin")
		if _, err = os.Stat(binPath); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		files, err := os.ReadDir(binPath)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, file := range files {
			if _, alreadySet := bins[file.Name()]; !alreadySet {
				bins[file.Name()] = filepath.Join(binPath, file.Name())
			}
		}
	}
	return lo.Values(bins), nil
}

func (d *Devbox) projectDirHash() string {
	hash, _ := cuecfg.Hash(d.projectDir)
	return hash
}

func (d *Devbox) addHashToEnv(env map[string]string) error {
	hash, err := cuecfg.Hash(env)
	if err == nil {
		env[d.ShellEnvHashKey()] = hash
	}
	return err
}

// parseEnvAndExcludeSpecialCases converts env as []string to map[string]string
// In case of pure shell, it leaks HOME and it leaks PATH with some modifications
func (d *Devbox) parseEnvAndExcludeSpecialCases(currentEnv []string) (map[string]string, error) {
	env := make(map[string]string, len(currentEnv))
	for _, kv := range currentEnv {
		key, val, found := strings.Cut(kv, "=")
		if !found {
			return nil, errors.Errorf("expected \"=\" in keyval: %s", kv)
		}
		if ignoreCurrentEnvVar[key] {
			continue
		}
		// handling special cases to for pure shell
		// Passing HOME for pure shell to leak through otherwise devbox binary won't work
		// We also include PATH to find the nix installation. It is cleaned for pure mode below
		if !d.pure || key == "HOME" || key == "PATH" {
			env[key] = val
		}
	}

	// handling special case for PATH
	if d.pure {
		// Finding nix executables in path and passing it through
		// As well as adding devbox itself to PATH
		// Both are needed for devbox commands inside pure shell to work
		includedInPath, err := findNixInPATH(env)
		if err != nil {
			return nil, err
		}
		includedInPath = append(includedInPath, dotdevboxBinPath(d))
		env["PATH"] = JoinPathLists(includedInPath...)
	}
	return env, nil
}

// ExportifySystemPathWithoutWrappers is a small utility to filter WrapperBin paths from PATH
func ExportifySystemPathWithoutWrappers() string {
	path := []string{}
	for _, p := range strings.Split(os.Getenv("PATH"), string(filepath.ListSeparator)) {
		// Intentionally do not include projectDir with plugin.WrapperBinPath so that
		// we filter out bin-wrappers for devbox-global and devbox-project.
		if !strings.Contains(p, plugin.WrapperBinPath) {
			path = append(path, p)
		}
	}

	envs := map[string]string{
		"PATH": strings.Join(path, string(filepath.ListSeparator)),
	}

	return exportify(envs)
}

func (d *Devbox) PluginManager() *plugin.Manager {
	return d.pluginManager
}

func (d *Devbox) Lockfile() *lock.File {
	return d.lockfile
}
