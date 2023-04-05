// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package impl creates isolated development environments.
package impl

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/generate"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/conf"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/pkgslice"
	"go.jetpack.io/devbox/internal/planner"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/services"
	"go.jetpack.io/devbox/internal/telemetry"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/wrapnix"
)

const (
	// configFilename is name of the JSON file that defines a devbox environment.
	configFilename = "devbox.json"

	// shellHistoryFile keeps the history of commands invoked inside devbox shell
	shellHistoryFile = ".devbox/shell_history"

	scriptsDir           = ".devbox/gen/scripts"
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
	cfg *Config
	// projectDir is the directory where the config file (devbox.json) resides
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
		projectDir:    projectDir,
		pluginManager: plugin.NewManager(),
		writer:        writer,
	}
	return box, nil
}

func (d *Devbox) ProjectDir() string {
	return d.projectDir
}

func (d *Devbox) Config() *Config {
	return d.cfg
}

func (d *Devbox) ShellPlan() (*plansdk.ShellPlan, error) {
	shellPlan := planner.GetShellPlan(d.projectDir, d.mergedPackages())
	shellPlan.DevPackages = pkgslice.Unique(append(d.localPackages(), shellPlan.DevPackages...))
	shellPlan.GlobalPackages = d.globalPackages()

	nixpkgsInfo, err := plansdk.GetNixpkgsInfo(d.cfg.Nixpkgs.Commit)
	if err != nil {
		return nil, err
	}
	shellPlan.NixpkgsInfo = nixpkgsInfo

	if len(shellPlan.GlobalPackages) > 0 {
		if globalHash := d.globalCommitHash(); globalHash != "" {
			globalNixpkgsInfo, err := plansdk.GetNixpkgsInfo(globalHash)
			if err != nil {
				return nil, err
			}
			shellPlan.GlobalNixpkgsInfo = globalNixpkgsInfo
		}
	}

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

	pluginHooks, err := plugin.InitHooks(d.mergedPackages(), d.projectDir)
	if err != nil {
		return err
	}

	env, err := d.cachedComputeNixEnv(ctx)
	if err != nil {
		return err
	}

	if err := wrapnix.CreateWrappers(ctx, d); err != nil {
		return err
	}

	shellStartTime := os.Getenv("DEVBOX_SHELL_START_TIME")
	if shellStartTime == "" {
		shellStartTime = telemetry.UnixTimestampFromTime(telemetry.CommandStartTime())
	}

	opts := []ShellOption{
		WithPluginInitHook(strings.Join(pluginHooks, "\n")),
		WithProfile(profileDir),
		WithHistoryFile(filepath.Join(d.projectDir, shellHistoryFile)),
		WithProjectDir(d.projectDir),
		WithEnvVariables(env),
		WithShellStartTime(shellStartTime),
	}

	shell, err := NewDevboxShell(d.cfg.Nixpkgs.Commit, opts...)
	if err != nil {
		return err
	}

	shell.UserInitHook = d.cfg.Shell.InitHook.String()
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

	env, err := d.cachedComputeNixEnv(ctx)
	if err != nil {
		return err
	}

	if err = wrapnix.CreateWrappers(ctx, d); err != nil {
		return err
	}

	var cmdWithArgs []string
	if _, ok := d.cfg.Shell.Scripts[cmdName]; ok {
		// it's a script, so replace the command with the script file's path.
		cmdWithArgs = append([]string{d.scriptPath(d.scriptFilename(cmdName))}, cmdArgs...)
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
		cmdWithArgs = []string{d.scriptPath(d.scriptFilename(arbitraryCmdFilename))}
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

func (d *Devbox) PrintEnv() (string, error) {
	ctx, task := trace.NewTask(context.Background(), "devboxPrintEnv")
	defer task.End()

	// generate in case user has old .devbox dir and is missing any files.
	if !d.isDotDevboxVersionCurrent() {
		if err := d.Generate(); err != nil {
			return "", err
		}
	}

	envs, err := d.cachedComputeNixEnv(ctx)
	if err != nil {
		return "", err
	}

	return exportify(envs), nil
}

func (d *Devbox) Info(pkg string, markdown bool) error {
	info, hasInfo := nix.PkgInfo(d.cfg.Nixpkgs.Commit, pkg)
	if !hasInfo {
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
	filesExist := plansdk.FileExists(devContainerJSONPath) || plansdk.FileExists(dockerfilePath)
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
	err = generate.CreateDevcontainer(devContainerPath, d.mergedPackages())
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
	filesExist := plansdk.FileExists(dockerfilePath)
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

		if strings.ToLower(result) == "y" || !isInteractiveMode {
			// .envrc file creation
			err := generate.CreateEnvrc(tmplFS, d.projectDir)
			if err != nil {
				return errors.WithStack(err)
			}
			cmd := exec.Command("direnv", "allow")
			err = cmd.Run()
			if err != nil {
				return errors.WithStack(err)
			}
		} else if source == "generate" {
			// .envrc file creation
			err := generate.CreateEnvrc(tmplFS, d.projectDir)
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
	svcSet := services.Services{}
	pluginSvcs, err := plugin.GetServices(d.mergedPackages(), d.projectDir)
	if err != nil {
		return svcSet, err
	}

	userSvcs := services.FromProcessComposeYaml(d.projectDir)

	return lo.Assign(pluginSvcs, userSvcs), nil

}

func (d *Devbox) StartServices(ctx context.Context, serviceNames ...string) error {
	if !IsDevboxShellEnabled() {
		return d.RunScript("devbox", append([]string{"services", "start"}, serviceNames...))
	}

	if !services.ProcessManagerIsRunning() {
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
			fmt.Printf("Error starting service %s: %s", s, err)
		} else {
			fmt.Printf("Service %s started successfully", s)
		}
	}
	return nil
}

func (d *Devbox) StopServices(ctx context.Context, serviceNames ...string) error {
	if !IsDevboxShellEnabled() {
		return d.RunScript("devbox", append([]string{"services", "stop"}, serviceNames...))
	}

	if !services.ProcessManagerIsRunning() {
		return usererr.New("Process manager is not running. Run `devbox services up` to start it.")
	}

	if len(serviceNames) == 0 {
		return services.StopProcessManager(ctx, d.writer)
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

func (d *Devbox) RestartServices(ctx context.Context, serviceNames ...string) error {
	if !IsDevboxShellEnabled() {
		return d.RunScript("devbox", append([]string{"services", "restart"}, serviceNames...))
	}

	if !services.ProcessManagerIsRunning() {
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

	// processCompose := services.LookupProcessCompose(d.projectDir, processComposeFileOrDir)

	processComposePath, err := utilityLookPath("process-compose")
	if err != nil {
		fmt.Fprintln(d.writer, "Installing process-compose. This may take a minute but will only happen once.")
		if err = d.addDevboxUtilityPackage("process-compose"); err != nil {
			return err
		}

		// re-lookup the path to process-compose
		processComposePath, err = utilityLookPath("process-compose")
		if err != nil {
			fmt.Fprintln(d.writer, "failed to find process-compose after installing it.")
			return err
		}
	}
	if !IsDevboxShellEnabled() {
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

	return services.StartProcessManager(ctx, requestedServices, svcs, d.projectDir, processComposePath, processComposeFileOrDir, background)
}

func (d *Devbox) generateShellFiles() error {
	plan, err := d.ShellPlan()
	if err != nil {
		return err
	}
	return generateForShell(d.projectDir, plan, d.pluginManager)
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
func (d *Devbox) computeNixEnv(ctx context.Context) (map[string]string, error) {
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
	originalPath, ok := env["DEVBOX_OG_PATH"]
	if !ok {
		env["DEVBOX_OG_PATH"] = currentEnvPath
		originalPath = currentEnvPath
	}

	vaf, err := nix.PrintDevEnv(ctx, d.nixShellFilePath(), d.nixFlakesFilePath())
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
	env["DEVBOX_SHELL_ENABLED"] = "1"      // Used to determine whether we're inside a shell (e.g. to prevent shell inception)

	debug.Log("nix environment PATH is: %s", env)

	// Add any vars defined in plugins.
	// TODO: Now that we have bin wrappers, this may can eventually be removed.
	// We still need to be able to add env variables to non-service binaries
	// (e.g. ruby). This would involve understanding what binaries are associated
	// to a given plugin.
	pluginEnv, err := plugin.Env(d.mergedPackages(), d.projectDir, env)
	if err != nil {
		return nil, err
	}

	addEnvIfNotPreviouslySetByDevbox(env, pluginEnv)

	// Prepend virtenv bin path first so user can override it if needed. Virtenv
	// is where the bin wrappers live
	env["PATH"] = JoinPathLists(filepath.Join(d.projectDir, plugin.WrapperBinPath), env["PATH"])

	// Include env variables in devbox.json
	configEnv := d.configEnvs(env)
	addEnvIfNotPreviouslySetByDevbox(env, configEnv)

	markEnvsAsSetByDevbox(pluginEnv, configEnv)

	nixEnvPath := env["PATH"]
	debug.Log("PATH after plugins and config is: %s", nixEnvPath)

	env["PATH"] = JoinPathLists(nixEnvPath, originalPath)
	debug.Log("computed environment PATH is: %s", env["PATH"])

	d.setCommonHelperEnvVars(env)

	return env, nil
}

var nixEnvCache map[string]string

// cachedComputeNixEnv is a wrapper around computeNixEnv that caches the result.
func (d *Devbox) cachedComputeNixEnv(ctx context.Context) (map[string]string, error) {
	var err error
	if nixEnvCache == nil {
		nixEnvCache, err = d.computeNixEnv(ctx)
	}
	return nixEnvCache, err
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
	pluginHooks, err := plugin.InitHooks(d.mergedPackages(), d.projectDir)
	if err != nil {
		return errors.WithStack(err)
	}
	hooks := strings.Join(append(pluginHooks, d.cfg.Shell.InitHook.String()), "\n\n")
	// always write it, even if there are no hooks, because scripts will source it.
	err = d.writeScriptFile(hooksFilename, hooks)
	if err != nil {
		return errors.WithStack(err)
	}
	written[d.scriptFilename(hooksFilename)] = struct{}{}

	// Write scripts to files.
	for name, body := range d.cfg.Shell.Scripts {
		err = d.writeScriptFile(name, d.scriptBody(body.String()))
		if err != nil {
			return errors.WithStack(err)
		}
		written[d.scriptFilename(name)] = struct{}{}
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
	script, err := os.Create(d.scriptPath(d.scriptFilename(name)))
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
	_, err = script.WriteString(body)
	return errors.WithStack(err)
}

func (d *Devbox) scriptPath(filename string) string {
	return filepath.Join(d.projectDir, scriptsDir, filename)
}

func (d *Devbox) scriptFilename(scriptName string) string {
	return scriptName + ".sh"
}

func (d *Devbox) scriptBody(body string) string {
	return fmt.Sprintf(". %s\n\n%s", d.scriptPath(d.scriptFilename(hooksFilename)), body)
}

func (d *Devbox) nixShellFilePath() string {
	return filepath.Join(d.projectDir, ".devbox/gen/shell.nix")
}

func (d *Devbox) nixFlakesFilePath() string {
	return filepath.Join(d.projectDir, ".devbox/gen/flake/flake.nix")
}

// mergedPackages returns the list of packages to be installed in the nix shell as
// specified by config and globals. It maintains the order of mergedPackages
// as specified by Config.Packages() (higher priority first)
func (d *Devbox) mergedPackages() []string {
	return d.cfg.Packages(d.writer)
}

func (d *Devbox) localPackages() []string {
	return d.cfg.RawPackages
}

func (d *Devbox) globalPackages() []string {
	dataPath, err := GlobalDataPath()
	if err != nil {
		ux.Ferror(d.writer, "unable to get devbox global data path: %s\n", err)
		return []string{}
	}
	global, err := readConfig(filepath.Join(dataPath, "devbox.json"))
	if err != nil {
		return []string{}
	}
	return global.RawPackages
}

func (d *Devbox) globalCommitHash() string {
	dataPath, err := GlobalDataPath()
	if err != nil {
		ux.Ferror(d.writer, "unable to get devbox global data path: %s\n", err)
		return ""
	}
	global, err := readConfig(filepath.Join(dataPath, "devbox.json"))
	if err != nil {
		return ""
	}
	return global.Nixpkgs.Commit
}

// configEnvs takes the computed env variables (nix + plugin) and adds env
// variables defined in Config. It also parses variables in config
// that are referenced by $VAR or ${VAR} and replaces them with
// their value in the computed env variables. Note, this doesn't
// allow env variables from outside the shell to be referenced so
// no leaked variables are caused by this function.
func (d *Devbox) configEnvs(computedEnv map[string]string) map[string]string {
	return conf.OSExpandEnvMap(d.cfg.Env, d.ProjectDir(), computedEnv)
}

// Move to a utility package?
func IsDevboxShellEnabled() bool {
	inDevboxShell, _ := strconv.ParseBool(os.Getenv("DEVBOX_SHELL_ENABLED"))
	return inDevboxShell
}

func commandExists(command string) bool {
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

// nix bins returns the paths to all the nix binaries that are installed by
// the flake. If there are conflicts, it returns the first one it finds of a
// give name. This matches how nix flakes behaves if there are conflicts in
// buildInputs
func (d *Devbox) NixBins(ctx context.Context) ([]string, error) {
	env, err := d.cachedComputeNixEnv(ctx)

	if err != nil {
		return nil, err
	}
	dirs := strings.Split(env["buildInputs"], " ")
	bins := map[string]string{}
	for _, dir := range dirs {
		binPath := filepath.Join(dir, "bin")
		if _, err = os.Stat(binPath); os.IsNotExist(err) {
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
