// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package impl creates isolated development environments.
package impl

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/impl/envpath"
	"go.jetpack.io/devbox/internal/impl/generate"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/shellgen"
	"go.jetpack.io/devbox/internal/telemetry"

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
	"go.jetpack.io/devbox/internal/services"
	"go.jetpack.io/devbox/internal/ux"
)

const (

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"

	arbitraryCmdFilename = ".cmd"
)

type Devbox struct {
	cfg                      *devconfig.Config
	env                      map[string]string
	lockfile                 *lock.File
	nix                      nix.Nixer
	projectDir               string
	pluginManager            *plugin.Manager
	preservePathStack        bool
	pure                     bool
	allowInsecureAdds        bool
	customProcessComposeFile string
	OmitBinWrappersFromPath  bool

	// This is needed because of the --quiet flag.
	stderr io.Writer
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
		cfg:                      cfg,
		env:                      opts.Env,
		nix:                      &nix.Nix{},
		projectDir:               projectDir,
		pluginManager:            plugin.NewManager(),
		stderr:                   opts.Stderr,
		preservePathStack:        opts.PreservePathStack,
		pure:                     opts.Pure,
		customProcessComposeFile: opts.CustomProcessComposeFile,
		allowInsecureAdds:        opts.AllowInsecureAdds,
		OmitBinWrappersFromPath:  opts.OmitBinWrappersFromPath,
	}

	lock, err := lock.GetFile(box)
	if err != nil {
		return nil, err
	}
	// if lockfile has any allow insecure, we need to set the env var to ensure
	// all nix commands work.
	if opts.AllowInsecureAdds || lock.HasAllowInsecurePackages() {
		nix.AllowInsecurePackages()
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
	pkgHashes := lo.Map(d.configPackages(), func(i *devpkg.Package, _ int) string { return i.Hash() })
	includeHashes := lo.Map(d.Includes(), func(i plugin.Includable, _ int) string { return i.Hash() })
	h, err := d.cfg.Hash()
	if err != nil {
		return "", err
	}
	return cuecfg.Hash(
		h + strings.Join(pkgHashes, "") + strings.Join(includeHashes, ""),
	)
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

	profileDir, err := d.profilePath()
	if err != nil {
		return err
	}

	envs, err := d.ensurePackagesAreInstalledAndComputeEnv(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintln(d.stderr, "Starting a devbox shell...")

	// Used to determine whether we're inside a shell (e.g. to prevent shell inception)
	// TODO: This is likely obsolete but we need to decide what happens when
	// the user does shell-ception. One option is to leave the current shell and
	// join a new one (that way they are not in nested shells.)
	envs[envir.DevboxShellEnabled] = "1"

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

	if err := shellgen.WriteScriptsToFiles(d); err != nil {
		return err
	}

	env, err := d.ensurePackagesAreInstalledAndComputeEnv(ctx)
	if err != nil {
		return err
	}
	// Used to determine whether we're inside a shell (e.g. to prevent shell inception)
	// This is temporary because StartServices() needs it but should be replaced with
	// better alternative since devbox run and devbox shell are not the same.
	env["DEVBOX_SHELL_ENABLED"] = "1"

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

	return d.ensurePackagesAreInstalled(ctx, ensure)
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

func (d *Devbox) NixEnv(ctx context.Context, opts devopt.NixEnvOpts) (string, error) {
	ctx, task := trace.NewTask(ctx, "devboxNixEnv")
	defer task.End()

	var envs map[string]string
	var err error

	if opts.DontRecomputeEnvironment {
		upToDate, _ := d.lockfile.IsUpToDateAndInstalled()
		if !upToDate {
			cmd := `eval "$(devbox global shellenv --recompute)"`
			if strings.HasSuffix(os.Getenv("SHELL"), "fish") {
				cmd = `devbox global shellenv --recompute | source`
			}
			ux.Finfo(
				d.stderr,
				"Your devbox environment may be out of date. Please run \n\n%s\n\n",
				cmd,
			)
		}

		envs, err = d.computeNixEnv(ctx, true /*usePrintDevEnvCache*/)
	} else {
		envs, err = d.ensurePackagesAreInstalledAndComputeEnv(ctx)
	}

	if err != nil {
		return "", err
	}

	envStr := exportify(envs)

	if opts.RunHooks {
		hooksStr := ". " + shellgen.ScriptPath(d.ProjectDir(), shellgen.HooksFilename)
		envStr = fmt.Sprintf("%s\n%s;\n", envStr, hooksStr)
	}

	return envStr, nil
}

func (d *Devbox) EnvVars(ctx context.Context) ([]string, error) {
	ctx, task := trace.NewTask(ctx, "devboxEnvVars")
	defer task.End()
	// this only returns env variables for the shell environment excluding hooks
	envs, err := d.ensurePackagesAreInstalledAndComputeEnv(ctx)
	if err != nil {
		return nil, err
	}
	return envir.MapToPairs(envs), nil
}

func (d *Devbox) shellEnvHashKey() string {
	// Don't make this a const so we don't use it by itself accidentally
	return "__DEVBOX_SHELLENV_HASH_" + d.projectDirHash()
}

func (d *Devbox) Info(ctx context.Context, pkg string, markdown bool) (string, error) {
	ctx, task := trace.NewTask(ctx, "devboxInfo")
	defer task.End()

	name, version, isVersioned := searcher.ParseVersionedPackage(pkg)
	if !isVersioned {
		name = pkg
		version = "latest"
	}

	packageVersion, err := searcher.Client().Resolve(name, version)
	if err != nil {
		if !errors.Is(err, searcher.ErrNotFound) {
			return "", usererr.WithUserMessage(err, "Package %q not found\n", pkg)
		}

		packageVersion = nil
		// fallthrough to below
	}

	if packageVersion == nil {
		return "", usererr.WithUserMessage(err, "Package %q not found\n", pkg)
	}

	// we should only have one result
	info := fmt.Sprintf(
		"%s%s %s\n%s\n",
		lo.Ternary(markdown, "## ", ""),
		packageVersion.Name,
		packageVersion.Version,
		packageVersion.Summary,
	)
	readme, err := plugin.Readme(
		ctx,
		devpkg.PackageFromString(pkg, d.lockfile),
		d.projectDir,
		markdown,
	)
	if err != nil {
		return "", err
	}
	return info + readme, nil
}

// GenerateDevcontainer generates devcontainer.json and Dockerfile for vscode run-in-container
// and GitHub Codespaces
func (d *Devbox) GenerateDevcontainer(ctx context.Context, generateOpts devopt.GenerateOpts) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerateDevcontainer")
	defer task.End()

	// construct path to devcontainer directory
	devContainerPath := filepath.Join(d.projectDir, ".devcontainer/")
	devContainerJSONPath := filepath.Join(devContainerPath, "devcontainer.json")
	dockerfilePath := filepath.Join(devContainerPath, "Dockerfile")

	// check if devcontainer.json or Dockerfile exist
	filesExist := fileutil.Exists(devContainerJSONPath) || fileutil.Exists(dockerfilePath)
	if !generateOpts.Force && filesExist {
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

	// Setup generate parameters
	gen := &generate.Options{
		Path:           devContainerPath,
		RootUser:       generateOpts.RootUser,
		IsDevcontainer: true,
		Pkgs:           d.PackageNames(),
		LocalFlakeDirs: d.getLocalFlakesDirs(),
	}

	// generate dockerfile
	err = gen.CreateDockerfile(ctx)
	if err != nil {
		return redact.Errorf("error generating dev container Dockerfile in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	// generate devcontainer.json
	err = gen.CreateDevcontainer(ctx)
	if err != nil {
		return redact.Errorf("error generating devcontainer.json in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	return nil
}

// GenerateDockerfile generates a Dockerfile that replicates the devbox shell
func (d *Devbox) GenerateDockerfile(ctx context.Context, generateOpts devopt.GenerateOpts) error {
	ctx, task := trace.NewTask(ctx, "devboxGenerateDockerfile")
	defer task.End()

	dockerfilePath := filepath.Join(d.projectDir, "Dockerfile")
	// check if Dockerfile doesn't exist
	filesExist := fileutil.Exists(dockerfilePath)
	if !generateOpts.Force && filesExist {
		return usererr.New(
			"Dockerfile is already present in the current directory. " +
				"Remove it or use --force to overwrite it.",
		)
	}

	// Setup Generate parameters
	gen := &generate.Options{
		Path:           d.projectDir,
		RootUser:       generateOpts.RootUser,
		IsDevcontainer: false,
		Pkgs:           d.PackageNames(),
		LocalFlakeDirs: d.getLocalFlakesDirs(),
	}

	// generate dockerfile
	return errors.WithStack(gen.CreateDockerfile(ctx))
}

func PrintEnvrcContent(w io.Writer, envFlags devopt.EnvFlags) error {
	return generate.EnvrcContent(w, envFlags)
}

// GenerateEnvrcFile generates a .envrc file that makes direnv integration convenient
func (d *Devbox) GenerateEnvrcFile(ctx context.Context, force bool, envFlags devopt.EnvFlags) error {
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
		d.stderr, func() *bool { return lo.ToPtr(false) },
	); err != nil {
		return err
	}

	// generate all shell files to ensure we can refer to them in the .envrc script
	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	// .envrc file creation
	err := generate.CreateEnvrc(ctx, d.projectDir, envFlags)
	if err != nil {
		return errors.WithStack(err)
	}
	ux.Fsuccess(d.stderr, "generated .envrc file\n")
	if cmdutil.Exists("direnv") {
		cmd := exec.Command("direnv", "allow")
		err := cmd.Run()
		if err != nil {
			return errors.WithStack(err)
		}
		ux.Fsuccess(d.stderr, "ran `direnv allow`\n")
	}
	return nil
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	return d.cfg.SaveTo(d.ProjectDir())
}

func (d *Devbox) Services() (services.Services, error) {
	pluginSvcs, err := d.pluginManager.GetServices(d.InstallablePackages(), d.cfg.Include)
	if err != nil {
		return nil, err
	}

	userSvcs := services.FromUserProcessCompose(d.projectDir, d.customProcessComposeFile)

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
		fmt.Fprintln(d.stderr, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.stderr, "\nNOTE: We recommend using `devbox services up` to start process-compose and your services")
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
		err := services.StartServices(ctx, d.stderr, s, d.projectDir)
		if err != nil {
			fmt.Fprintf(d.stderr, "Error starting service %s: %s", s, err)
		} else {
			fmt.Fprintf(d.stderr, "Service %s started successfully", s)
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
			return usererr.New(fmt.Sprintf("Service %s not found in your project", s))
		}
		err := services.StopServices(ctx, s, d.projectDir, d.stderr)
		if err != nil {
			fmt.Fprintf(d.stderr, "Error stopping service %s: %s", s, err)
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
		fmt.Fprintln(d.stderr, "Process-compose is not running. Starting it now...")
		fmt.Fprintln(d.stderr, "\nTip: We recommend using `devbox services up` to start process-compose and your services")
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
		err := services.RestartServices(ctx, s, d.projectDir, d.stderr)
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
		fmt.Fprintln(d.stderr, "Installing process-compose. This may take a minute but will only happen once.")
		if err = d.addDevboxUtilityPackage(ctx, "github:F1bonacc1/process-compose/v0.43.1"); err != nil {
			return err
		}

		// re-lookup the path to process-compose
		processComposePath, err = utilityLookPath("process-compose")
		if err != nil {
			fmt.Fprintln(d.stderr, "failed to find process-compose after installing it.")
			return err
		}
	}

	// Start the process manager

	return services.StartProcessManager(
		ctx,
		d.stderr,
		requestedServices,
		svcs,
		d.projectDir,
		processComposePath,
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
// TODO: Rename to computeDevboxEnv?
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

	debug.Log("current environment PATH is: %s", env["PATH"])

	originalEnv := make(map[string]string, len(env))
	maps.Copy(originalEnv, env)

	vaf, err := d.nix.PrintDevEnv(ctx, &nix.PrintDevEnvArgs{
		FlakeDir:             d.flakeDir(),
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
	pluginEnv, err := d.pluginManager.Env(d.InstallablePackages(), d.cfg.Include, env)
	if err != nil {
		return nil, err
	}

	addEnvIfNotPreviouslySetByDevbox(env, pluginEnv)

	env["PATH"] = envpath.JoinPathLists(
		nix.ProfileBinPath(d.projectDir),
		env["PATH"],
	)

	if !d.OmitBinWrappersFromPath {
		env["PATH"] = envpath.JoinPathLists(
			filepath.Join(d.projectDir, plugin.WrapperBinPath),
			env["PATH"],
		)
	}

	env["PATH"], err = d.addUtilitiesToPath(ctx, env["PATH"])
	if err != nil {
		return nil, err
	}

	// Include env variables in devbox.json
	configEnv, err := d.configEnvs(ctx, env)
	if err != nil {
		return nil, err
	}
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

	runXPaths, err := d.RunXPaths(ctx)
	if err != nil {
		return nil, err
	}
	nixEnvPath = envpath.JoinPathLists(nixEnvPath, runXPaths)

	pathStack := envpath.Stack(env, originalEnv)
	pathStack.Push(env, d.projectDirHash(), nixEnvPath, d.preservePathStack)
	env["PATH"] = pathStack.Path(env)
	debug.Log("New path stack is: %s", pathStack)

	debug.Log("computed environment PATH is: %s", env["PATH"])

	d.setCommonHelperEnvVars(env)

	if !d.pure {
		// preserve the original XDG_DATA_DIRS by prepending to it
		env["XDG_DATA_DIRS"] = envpath.JoinPathLists(env["XDG_DATA_DIRS"], os.Getenv("XDG_DATA_DIRS"))
	}

	for k, v := range d.env {
		env[k] = v
	}

	return env, d.addHashToEnv(env)
}

// ensurePackagesAreInstalledAndComputeEnv does what it says :P
func (d *Devbox) ensurePackagesAreInstalledAndComputeEnv(
	ctx context.Context,
) (map[string]string, error) {
	defer debug.FunctionTimer().End()

	// When ensurePackagesAreInstalled is called with ensure=true, it always
	// returns early if the lockfile is up to date. So we don't need to check here
	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return nil, err
	}

	// Since ensurePackagesAreInstalled calls computeNixEnv when not up do date,
	// it's ok to use usePrintDevEnvCache=true here always. This does end up
	// doing some non-nix work twice if lockfile is not up to date.
	// TODO: Improve this to avoid extra work.
	return d.computeNixEnv(ctx, true)
}

func (d *Devbox) nixPrintDevEnvCachePath() string {
	return filepath.Join(d.projectDir, ".devbox/.nix-print-dev-env-cache")
}

func (d *Devbox) flakeDir() string {
	return filepath.Join(d.projectDir, ".devbox/gen/flake")
}

// ConfigPackageNames returns the package names as defined in devbox.json
func (d *Devbox) PackageNames() []string {
	// TODO savil: centralize implementation by calling d.configPackages and getting pkg.Raw
	// Skipping for now to avoid propagating the error value.
	return d.cfg.Packages.VersionedNames()
}

// configPackages returns the packages that are defined in devbox.json
// NOTE: the return type is different from devconfig.Packages
func (d *Devbox) configPackages() []*devpkg.Package {
	return devpkg.PackagesFromConfig(d.cfg, d.lockfile)
}

// InstallablePackages returns the packages that are to be installed
func (d *Devbox) InstallablePackages() []*devpkg.Package {
	return lo.Filter(d.configPackages(), func(pkg *devpkg.Package, _ int) bool {
		return pkg.IsInstallable()
	})
}

// AllInstallablePackages returns installable user packages and plugin
// packages concatenated in correct order
func (d *Devbox) AllInstallablePackages() ([]*devpkg.Package, error) {
	userPackages := d.InstallablePackages()
	return d.PluginManager().ProcessPluginPackages(userPackages)
}

func (d *Devbox) Includes() []plugin.Includable {
	includes := []plugin.Includable{}
	for _, includePath := range d.cfg.Include {
		if include, err := d.pluginManager.ParseInclude(includePath); err == nil {
			includes = append(includes, include)
		}
	}
	return includes
}

func (d *Devbox) HasDeprecatedPackages() bool {
	for _, pkg := range d.configPackages() {
		if pkg.IsLegacy() {
			return true
		}
	}
	return false
}

func (d *Devbox) findPackageByName(name string) (*devpkg.Package, error) {
	if name == "" {
		return nil, errors.New("package name cannot be empty")
	}
	results := map[*devpkg.Package]bool{}
	for _, pkg := range d.configPackages() {
		if pkg.Raw == name || pkg.CanonicalName() == name {
			results[pkg] = true
		}
	}
	if len(results) > 1 {
		return nil, usererr.New(
			"found multiple packages with name %s: %s. Please specify version",
			name,
			lo.Keys(results),
		)
	}
	if len(results) == 0 {
		return nil, usererr.WithUserMessage(
			searcher.ErrNotFound, "no package found with name %s", name)
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
				d.stderr,
				"Your .envrc file seems to be out of date. "+
					"Run `devbox generate direnv --force` to update it.\n"+
					"Or silence this warning by setting DEVBOX_NO_ENVRC_UPDATE=1 env variable.\n",
			)
		}
	}
	return nil
}

// configEnvs takes the existing environment (nix + plugin) and adds env
// variables defined in Config. It also parses variables in config
// that are referenced by $VAR or ${VAR} and replaces them with
// their value in the existing env variables. Note, this doesn't
// allow env variables from outside the shell to be referenced so
// no leaked variables are caused by this function.
func (d *Devbox) configEnvs(
	ctx context.Context,
	existingEnv map[string]string,
) (map[string]string, error) {
	env, err := d.cfg.ComputedEnv(ctx, d.ProjectDir())
	if err != nil {
		return nil, err
	}
	return conf.OSExpandEnvMap(env, existingEnv, d.ProjectDir()), nil
}

// ignoreCurrentEnvVar contains environment variables that Devbox should remove
// from the slice of [os.Environ] variables before sourcing them. These are
// variables that are set automatically by a new shell.
var ignoreCurrentEnvVar = map[string]bool{
	envir.DevboxLatestVersion: true,

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
	profileLibDir := filepath.Join(d.projectDir, nix.ProfilePath, "lib")
	env["LD_LIBRARY_PATH"] = envpath.JoinPathLists(profileLibDir, env["LD_LIBRARY_PATH"])
	env["LIBRARY_PATH"] = envpath.JoinPathLists(profileLibDir, env["LIBRARY_PATH"])
}

// nixBins returns the paths to all the nix binaries that are installed by
// the flake. If there are conflicts, it returns the first one it finds of a
// give name. This matches how nix flakes behaves if there are conflicts in
// buildInputs
func (d *Devbox) nixBins(env map[string]string) ([]string, error) {
	dirs := strings.Split(env["buildInputs"], " ")
	bins := map[string]string{}
	for _, dir := range dirs {
		binPath := filepath.Join(dir, "bin")
		if _, err := os.Stat(binPath); errors.Is(err, fs.ErrNotExist) {
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
		env[d.shellEnvHashKey()] = hash
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
		// TERM leaking through is to enable colored text in the pure shell
		if !d.pure || key == "HOME" || key == "PATH" || key == "TERM" {
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
		env["PATH"] = envpath.JoinPathLists(includedInPath...)
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

func (d *Devbox) RunXPaths(ctx context.Context) (string, error) {
	packages := lo.Filter(d.InstallablePackages(), devpkg.IsRunX)
	paths := []string{}
	for _, pkg := range packages {
		lockedPkg, err := d.lockfile.Resolve(pkg.Raw)
		if err != nil {
			return "", err
		}
		p, err := pkgtype.RunXClient().Install(ctx, lockedPkg.Resolved)
		if err != nil {
			return "", err
		}
		paths = append(paths, p...)
	}
	return envpath.JoinPathLists(paths...), nil
}
