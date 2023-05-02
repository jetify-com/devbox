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
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/generate"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/conf"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/services"
	"go.jetpack.io/devbox/internal/telemetry"
	"go.jetpack.io/devbox/internal/wrapnix"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"

	scriptsDir = ".devbox/gen/scripts"

	// hooksFilename is the name of the file that contains the project's init-hooks and plugin hooks
	hooksFilename        = ".hooks"
	arbitraryCmdFilename = ".cmd"
)

func InitConfig(dir string, writer io.Writer) (created bool, err error) {
	cfgPath := filepath.Join(dir, configFilename)

	config := &Config{
		Nixpkgs: NixpkgsConfig{
			Commit: plansdk.DefaultNixpkgsCommit,
		},
	}
	if featureflag.EnvConfig.Enabled() {
		// TODO: after removing feature flag we can decide if we want
		// to have omitempty for Env in Config or not.
		config.Env = map[string]string{}
	}
	// package suggestion
	pkgsToSuggest, err := initrec.Get(dir)
	if err != nil {
		return false, err
	}
	if len(pkgsToSuggest) > 0 {
		s := fmt.Sprintf("devbox add %s", strings.Join(pkgsToSuggest, " "))
		fmt.Fprintf(
			writer,
			"We detected extra packages you may need. To install them, run `%s`\n",
			color.HiYellowString(s),
		)
	}

	return cuecfg.InitFile(cfgPath, config)
}

type Devbox struct {
	cfg           *Config
	lockfile      *lock.File
	nix           nix.Nixer
	projectDir    string
	pluginManager *plugin.Manager
	writer        io.Writer
}

func Open(path string, writer io.Writer) (*Devbox, error) {
	projectDir, err := findProjectDir(path)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(projectDir, configFilename)

	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = upgradeConfig(cfg, cfgPath); err != nil {
		return nil, err
	}

	box := &Devbox{
		cfg:           cfg,
		nix:           &nix.Nix{},
		projectDir:    projectDir,
		pluginManager: plugin.NewManager(),
		writer:        writer,
	}
	lock, err := lock.GetFile(box, searcher.Client())
	if err != nil {
		return nil, err
	}
	box.lockfile = lock
	return box, nil
}

func (d *Devbox) ProjectDir() string {
	return d.projectDir
}

func (d *Devbox) Config() *Config {
	return d.cfg
}

func (d *Devbox) ConfigHash() (string, error) {
	return d.cfg.Hash()
}

func (d *Devbox) NixPkgsCommitHash() string {
	return d.cfg.Nixpkgs.Commit
}

func (d *Devbox) ShellPlan() (*plansdk.ShellPlan, error) {
	shellPlan := planner.GetShellPlan(d.projectDir, d.packages())
	shellPlan.FlakeInputs = d.flakeInputs()

	nixpkgsInfo := plansdk.GetNixpkgsInfo(d.cfg.Nixpkgs.Commit)
	shellPlan.NixpkgsInfo = nixpkgsInfo

	return shellPlan, nil
}

func (d *Devbox) Generate() error {
	return errors.WithStack(d.generateShellFiles())
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

	shellStartTime := os.Getenv(envir.DevboxShellStartTime)
	if shellStartTime == "" {
		shellStartTime = telemetry.UnixTimestampFromTime(telemetry.CommandStartTime())
	}

	opts := []ShellOption{
		WithHooksFilePath(d.scriptPath(hooksFilename)),
		WithProfile(profileDir),
		WithHistoryFile(filepath.Join(d.projectDir, shellHistoryFile)),
		WithProjectDir(d.projectDir),
		WithEnvVariables(envs),
		WithShellStartTime(shellStartTime),
	}

	shell, err := NewDevboxShell(d.cfg.Nixpkgs.Commit, opts...)
	if err != nil {
		return err
	}

	return shell.Run()
}

func (d *Devbox) RunScript(cmdName string, cmdArgs []string) error {
	ctx, task := trace.NewTask(context.Background(), "devboxRun")
	defer task.End()

	if err := d.ensurePackagesAreInstalled(ctx, ensure); err != nil {
		return err
	}

	if err := d.writeScriptsToFiles(); err != nil {
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

	var cmdWithArgs []string
	if _, ok := d.cfg.Shell.Scripts[cmdName]; ok {
		// it's a script, so replace the command with the script file's path.
		cmdWithArgs = append([]string{d.scriptPath(cmdName)}, cmdArgs...)
	} else {
		// Arbitrary commands should also run the hooks, so we write them to a file as well. However, if the
		// command args include env variable evaluations, then they'll be evaluated _before_ the hooks run,
		// which we don't want. So, one solution is to write the entire command and its arguments into the
		// file itself, but that may not be great if the variables contain sensitive information. Instead,
		// we save the entire command (with args) into the DEVBOX_RUN_CMD var, and then the script evals it.
		err := d.writeScriptFile(arbitraryCmdFilename, d.scriptBody("eval $DEVBOX_RUN_CMD\n"))
		if err != nil {
			return err
		}
		cmdWithArgs = []string{d.scriptPath(arbitraryCmdFilename)}
		env["DEVBOX_RUN_CMD"] = strings.Join(append([]string{cmdName}, cmdArgs...), " ")
	}

	return nix.RunScript(d.projectDir, strings.Join(cmdWithArgs, " "), env)
}

func (d *Devbox) ListScripts() []string {
	keys := make([]string, len(d.cfg.Shell.Scripts))
	i := 0
	for k := range d.cfg.Shell.Scripts {
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
		hooksStr := ". " + d.scriptPath(hooksFilename)
		envStr = fmt.Sprintf("%s\n%s;\n", envStr, hooksStr)
	}

	return envStr, nil
}

func (d *Devbox) ShellEnvHash(ctx context.Context) (string, error) {
	envs, err := d.nixEnv(ctx)
	if err != nil {
		return "", err
	}

	return envs[devboxShellEnvHashVarName], nil
}

func (d *Devbox) Info(pkg string, markdown bool) error {
	info := nix.PkgInfo(d.cfg.Nixpkgs.Commit, pkg)
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
		pkg,
		d.projectDir,
		d.writer,
		markdown,
	)
}

// GenerateDevcontainer generates devcontainer.json and Dockerfile for vscode run-in-container
// and GitHub Codespaces
func (d *Devbox) GenerateDevcontainer(force bool) error {
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
	err = generate.CreateDockerfile(tmplFS, devContainerPath)
	if err != nil {
		return redact.Errorf("error generating dev container Dockerfile in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	// generate devcontainer.json
	err = generate.CreateDevcontainer(devContainerPath, d.packages())
	if err != nil {
		return redact.Errorf("error generating devcontainer.json in <project>/%s: %w",
			redact.Safe(filepath.Base(devContainerPath)), err)
	}
	return nil
}

// GenerateDockerfile generates a Dockerfile that replicates the devbox shell
func (d *Devbox) GenerateDockerfile(force bool) error {
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
	return errors.WithStack(generate.CreateDockerfile(tmplFS, d.projectDir))
}

// GenerateEnvrc generates a .envrc file that makes direnv integration convenient
func (d *Devbox) GenerateEnvrc(force bool, source string) error {
	ctx, task := trace.NewTask(context.Background(), "devboxGenerateEnvrc")
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
	if commandExists("direnv") {
		// prompt for direnv allow
		var result string
		isInteractiveMode := isatty.IsTerminal(os.Stdin.Fd())
		if isInteractiveMode {
			prompt := &survey.Input{
				Message: "Do you want to enable direnv integration for this devbox project? [y/N]",
			}
			err := survey.AskOne(prompt, &result)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		if strings.ToLower(result) == "y" || !isInteractiveMode || source == "generate" {
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
			err := generate.CreateEnvrc(tmplFS, d.projectDir)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		if strings.ToLower(result) == "y" || !isInteractiveMode {
			cmd := exec.Command("direnv", "allow")
			err := cmd.Run()
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// saveCfg writes the config file to the devbox directory.
func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.projectDir, configFilename)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}

func (d *Devbox) Services() (services.Services, error) {
	pluginSvcs, err := plugin.GetServices(d.packages(), d.projectDir)
	if err != nil {
		return nil, err
	}

	userSvcs := services.FromProcessComposeYaml(d.projectDir)

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
	if !envir.IsDevboxShellEnabled() {
		return d.RunScript("devbox", append([]string{"services", "start"}, serviceNames...))
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
	if !envir.IsDevboxShellEnabled() {
		args := []string{"services", "stop"}
		args = append(args, serviceNames...)
		if allProjects {
			args = append(args, "--all-projects")
		}
		return d.RunScript("devbox", args)
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
	if !envir.IsDevboxShellEnabled() {
		return d.RunScript("devbox", []string{"services", "ls"})
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
	if !envir.IsDevboxShellEnabled() {
		return d.RunScript("devbox", append([]string{"services", "restart"}, serviceNames...))
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
	if !envir.IsDevboxShellEnabled() {
		args := []string{"services", "up"}
		args = append(args, requestedServices...)
		if processComposeFileOrDir != "" {
			args = append(args, "--process-compose-file", processComposeFileOrDir)
		}
		if background {
			args = append(args, "--background")
		}
		return d.RunScript("devbox", args)
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

	currentEnv := os.Environ()
	env := make(map[string]string, len(currentEnv))
	for _, kv := range currentEnv {
		key, val, found := strings.Cut(kv, "=")
		if !found {
			return nil, errors.Errorf("expected \"=\" in keyval: %s", kv)
		}
		if ignoreCurrentEnvVar[key] {
			continue
		}
		env[key] = val
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
	pluginEnv, err := plugin.Env(d.packages(), d.projectDir, env)
	if err != nil {
		return nil, err
	}

	addEnvIfNotPreviouslySetByDevbox(env, pluginEnv)

	// Prepend virtenv bin path first so user can override it if needed. Virtenv
	// is where the bin wrappers live
	env["PATH"] = JoinPathLists(
		filepath.Join(d.projectDir, plugin.WrapperBinPath),
		// Adding profile bin path is a temporary hack. Some packages .e.g. curl
		// don't export the correct bin in the package, instead they export
		// as a propagated build input. This can be fixed in 2 ways:
		// * have NixBins() recursively look for bins in propagated build inputs
		// * Turn existing planners into flakes (i.e. php, haskell) and use the bins
		// in the profile.
		// Landau: I prefer option 2 because it doesn't require us to re-implement
		// nix recursive bin lookup.
		nix.ProfileBinPath(d.projectDir),
		env["PATH"],
	)

	// Include env variables in devbox.json
	configEnv := d.configEnvs(env)
	addEnvIfNotPreviouslySetByDevbox(env, configEnv)

	markEnvsAsSetByDevbox(pluginEnv, configEnv)

	nixEnvPath := env["PATH"]
	debug.Log("PATH after plugins and config is: %s", nixEnvPath)

	env["PATH"] = JoinPathLists(nixEnvPath, originalPath)
	debug.Log("computed environment PATH is: %s", env["PATH"])

	d.setCommonHelperEnvVars(env)

	return env, addHashToEnv(env)
}

var nixEnvCache map[string]string

// nixEnv is a wrapper around computeNixEnv that caches the result.
// Note that this is in-memory cache of the final environment, and not the same
// as the nix print-dev-env cache which is stored in a file.
func (d *Devbox) nixEnv(ctx context.Context) (map[string]string, error) {
	var err error
	if nixEnvCache == nil {
		usePrintDevEnvCache := false

		// If lockfile is up-to-date, we can use the print-dev-env cache.
		if lock, err := lock.Local(d); err != nil {
			return nil, err
		} else if upToDate, err := lock.IsUpToDate(); err != nil {
			return nil, err
		} else if upToDate {
			usePrintDevEnvCache = true
		}

		nixEnvCache, err = d.computeNixEnv(ctx, usePrintDevEnvCache)
	}
	return nixEnvCache, err
}

func (d *Devbox) ogPathKey() string {
	return "DEVBOX_OG_PATH_" + d.projectDirHash()
}

// writeScriptsToFiles writes scripts defined in devbox.json into files inside .devbox/gen/scripts.
// Scripts (and hooks) are persisted so that we can easily call them from devbox run (inside or outside shell).
func (d *Devbox) writeScriptsToFiles() error {
	err := os.MkdirAll(filepath.Join(d.projectDir, scriptsDir), 0755) // Ensure directory exists.
	if err != nil {
		return errors.WithStack(err)
	}

	// Read dir contents before writing, so we can clean up later.
	entries, err := os.ReadDir(filepath.Join(d.projectDir, scriptsDir))
	if err != nil {
		return errors.WithStack(err)
	}

	// Write all hooks to a file.
	written := map[string]struct{}{} // set semantics; value is irrelevant
	pluginHooks, err := plugin.InitHooks(d.packages(), d.projectDir)
	if err != nil {
		return errors.WithStack(err)
	}
	hooks := strings.Join(append(pluginHooks, d.cfg.Shell.InitHook.String()), "\n\n")
	// always write it, even if there are no hooks, because scripts will source it.
	err = d.writeScriptFile(hooksFilename, hooks)
	if err != nil {
		return errors.WithStack(err)
	}
	written[d.scriptPath(hooksFilename)] = struct{}{}

	// Write scripts to files.
	for name, body := range d.cfg.Shell.Scripts {
		err = d.writeScriptFile(name, d.scriptBody(body.String()))
		if err != nil {
			return errors.WithStack(err)
		}
		written[d.scriptPath(name)] = struct{}{}
	}

	// Delete any files that weren't written just now.
	for _, entry := range entries {
		if _, ok := written[entry.Name()]; !ok && !entry.IsDir() {
			err := os.Remove(d.scriptPath(entry.Name()))
			if err != nil {
				debug.Log("failed to clean up script file %s, error = %s", entry.Name(), err) // no need to fail run
			}
		}
	}

	return nil
}

func (d *Devbox) writeScriptFile(name string, body string) (err error) {
	script, err := os.Create(d.scriptPath(name))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		cerr := script.Close()
		if err == nil {
			err = cerr
		}
	}()
	err = script.Chmod(0755)
	if err != nil {
		return errors.WithStack(err)
	}

	if featureflag.ScriptExitOnError.Enabled() {
		body = fmt.Sprintf("set -e\n\n%s", body)
	}
	_, err = script.WriteString(body)
	return errors.WithStack(err)
}

func (d *Devbox) scriptPath(scriptName string) string {
	return scriptPath(d.projectDir, scriptName)
}

// scriptPath is a helper function, refactored out for use in tests.
// use `d.scriptPath` instead for production code.
func scriptPath(projectDir string, scriptName string) string {
	return filepath.Join(projectDir, scriptsDir, scriptName+".sh")
}

func (d *Devbox) scriptBody(body string) string {
	return fmt.Sprintf(". %s\n\n%s", d.scriptPath(hooksFilename), body)
}

func (d *Devbox) nixPrintDevEnvCachePath() string {
	return filepath.Join(d.projectDir, ".devbox/.nix-print-dev-env-cache")
}

func (d *Devbox) nixFlakesFilePath() string {
	return filepath.Join(d.projectDir, ".devbox/gen/flake/flake.nix")
}

// packages returns the list of packages to be installed in the nix shell.
func (d *Devbox) packages() []string {
	return d.cfg.Packages
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

func commandExists(command string) bool { // TODO: move to a utility package
	_, err := exec.LookPath(command)
	return err == nil
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

func addHashToEnv(env map[string]string) error {
	hash, err := cuecfg.Hash(env)
	if err == nil {
		env[devboxShellEnvHashVarName] = hash

	}
	return err
}
