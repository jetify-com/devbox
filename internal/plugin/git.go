package plugin

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/pkg/filecache"
)

var sshCache = filecache.New[[]byte]("devbox/plugin/ssh")
var gitCache = filecache.New[[]byte]("devbox/plugin/git")
var githubCache = filecache.New[[]byte]("devbox/plugin/github")
var gitlabCache = filecache.New[[]byte]("devbox/plugin/gitlab")
var bitbucketCache = filecache.New[[]byte]("devbox/plugin/bitbucket")

type gitPlugin struct {
	ref  flake.Ref
	name string
}

// Github only allows alphanumeric, hyphen, underscore, and period in repo names.
// but we clean up just in case.
var githubNameRegexp = regexp.MustCompile("[^a-zA-Z0-9-_.]+")

func newGitPlugin(ref flake.Ref) (*gitPlugin, error) {
	plugin := &gitPlugin{ref: ref}
	// For backward compatibility, we don't strictly require name to be present
	// in github plugins. If it's missing, we just use the directory as the name.
	name, err := getPluginNameFromContent(plugin)

	if err != nil && !errors.Is(err, errNameMissing) {
		return nil, err
	}

	if name == "" {
		name = strings.ReplaceAll(ref.Dir, "/", "-")
	}

	// gitlab repos can have up to 20 subgroups. We need to capture the subgroups in the plugin name
	repoDotted := strings.ReplaceAll(ref.Repo, "/", ".")

	plugin.name = githubNameRegexp.ReplaceAllString(
		strings.Join(lo.Compact([]string{ref.Owner, repoDotted, name}), "."),
		" ",
	)

	return plugin, nil
}

func (p *gitPlugin) Fetch() ([]byte, error) {
	content, err := p.FileContent(pluginConfigName)
	if err != nil {
		return nil, err
	}
	return jsonPurifyPluginContent(content)
}

func (p *gitPlugin) CanonicalName() string {
	return p.name
}

func (p *gitPlugin) Hash() string {
	return cachehash.Bytes([]byte(p.ref.String()))
}

func (p *gitPlugin) fetchSSHArchive(location string) ([]byte, error) {
	archiveDir, _ := os.MkdirTemp("", p.ref.Repo)
	archive := filepath.Join(archiveDir, p.ref.Owner+".tar.gz")
	args := strings.Fields(location + archive) // this is really just the base git archive command + file

	defer os.RemoveAll(archiveDir)

	cmd := exec.Command(args[0], args[1:]...)
	_, err := cmd.Output()

	if err != nil {
		slog.Error("Error executing git archive: " + err.Error())
		return nil, err
	}

	reader, err := os.Open(archive)
	err = fileutil.Untar(reader, archiveDir)

	if err != nil {
		slog.Error("Encountered error while trying to extract " + archive + ": " + err.Error())
		return nil, err
	}

	pluginJson := filepath.Join(archiveDir, p.ref.Dir, "plugin.json")
	file, err := os.Open(pluginJson)

	defer file.Close()
	info, err := file.Stat()

	if err != nil {
		slog.Error("Error extracting file " + file.Name() + ". Cannot process plugin.")
		return nil, err
	}

	if info.Size() == 0 {
		slog.Error("Extracted file " + file.Name() + " is empty. Cannot process plugin.")
		return nil, err
	}

	return io.ReadAll(file)
}

func (p *gitPlugin) fetchHttp(location string) ([]byte, error) {
	req, err := p.request(location)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, usererr.New(
			"failed to get plugin %s @ %s (Status code %d). \nPlease make "+
				"sure a plugin.json file exists in plugin directory.",
			p.LockfileKey(),
			req.URL.String(),
			res.StatusCode,
		)
	}

	return io.ReadAll(res.Body)
}

func (p *gitPlugin) FileContent(subpath string) ([]byte, error) {
	location, err := p.url(subpath)

	if err != nil {
		return nil, err
	}

	var bytes []byte

	if p.ref.Type == flake.TypeSSH {
		bytes, err = p.fetchSSHArchive(location)
	} else {
		bytes, err = p.fetchHttp(location)
	}

	if err != nil {
		return nil, err
	}

	process := func() ([]byte, time.Duration, error) {
		// Cache for 24 hours. Once we store the plugin in the lockfile, we
		// should cache this indefinitely and only invalidate if the plugin
		// is updated.
		return bytes, 24 * time.Hour, nil
	}

	switch p.ref.Type {
	case flake.TypeSSH:
		return sshCache.GetOrSet(location, process)
	case flake.TypeGitHub:
		return githubCache.GetOrSet(location, process)
	case flake.TypeGitLab:
		return gitlabCache.GetOrSet(location, process)
	case flake.TypeBitBucket:
		return bitbucketCache.GetOrSet(location, process)
	case flake.TypeGit:
		return gitCache.GetOrSet(location, process)
	default:
		slog.Error("Unable to handle flake ref type: " + p.ref.Type)
		return nil, err
	}
}

func (p *gitPlugin) url(subpath string) (string, error) {
	switch p.ref.Type {
	case flake.TypeSSH:
		return p.sshBaseGitCommand()
	case flake.TypeGit, flake.TypeGitHub, flake.TypeGitLab, flake.TypeBitBucket:
		return p.repoUrl(subpath)
	default:
		return "", errors.New("Unsupported plugin type: " + p.ref.Type)
	}
}

func (p *gitPlugin) sshBaseGitCommand() (string, error) {
	defaultBranch := "main"

	if p.ref.Host == flake.TypeGitHub+".com" {
		// using master for GitHub repos for the same reasoning established in `githubUrl`
		defaultBranch = "master"
	}

	prefix := "git archive --format=tar.gz --remote=ssh://git@"
	path, _ := url.JoinPath(p.ref.Owner, p.ref.Repo)
	branch := cmp.Or(p.ref.Rev, p.ref.Ref, defaultBranch)
	host := p.ref.Host

	// the Ref struct defaults the field to 0. This technically a valid port for UDP, but we aren't using UDP
	if p.ref.Port > 0 {
		host += ":" + fmt.Sprintf("%d", p.ref.Port)
	}

	command := fmt.Sprintf("%s%s/%s %s", prefix, host, path, branch)
	if p.ref.Dir != "" {
		command += fmt.Sprintf(" %s", p.ref.Dir)
	}
	command += " -o"

	slog.Debug("Generated base git archive command: " + command)
	return command, nil
}

func (p *gitPlugin) githubUrl(subpath string) (string, error) {
	// Github redirects "master" to "main" in new repos. They don't do the reverse
	// so setting master here is better.
	return url.JoinPath(
		"https://raw.githubusercontent.com/",
		p.ref.Owner,
		p.ref.Repo,
		cmp.Or(p.ref.Rev, p.ref.Ref, "master"),
		p.ref.Dir,
		subpath,
	)
}

func (p *gitPlugin) bitbucketUrl(subpath string) (string, error) {
	// bitbucket doesn't redirect master -> main or main -> master, so using "main"
	// as the default in this case

	return url.JoinPath(
		"https://api.bitbucket.org/2.0/repositories",
		p.ref.Owner,
		p.ref.Repo,
		"src",
		cmp.Or(p.ref.Rev, p.ref.Ref, "main"),
		p.ref.Dir,
		subpath,
	)
}

func (p *gitPlugin) genericGitUrl(subpath string) (string, error) {
	address, err := url.JoinPath(
		p.ref.Host,
		p.ref.Repo,
		cmp.Or(p.ref.Rev, p.ref.Ref, "main"),
		p.ref.Dir,
		subpath,
	)

	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(address)

	if err != nil {
		return "", err
	}

	query := parsed.Query()

	if p.ref.Dir != "" {
		query.Add("dir", p.ref.Dir)
	}

	if p.ref.Port != 0 {
		query.Add("port", fmt.Sprintf("%d", p.ref.Port))
	}

	// gitlab doesn't redirect master -> main or main -> master, so using "main"
	// as the default in this case
	query.Add("ref", cmp.Or(p.ref.Rev, p.ref.Ref, "main"))
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (p *gitPlugin) repoUrl(subpath string) (string, error) {
	if p.ref.Type == flake.TypeGitHub {
		return p.githubUrl(subpath)
	} else if p.ref.Type == flake.TypeGitLab {
		return p.gitlabUrl(subpath)
	} else if p.ref.Type == flake.TypeBitBucket {
		return p.bitbucketUrl(subpath)
	} else if p.ref.Type == flake.TypeGit {
		return p.genericGitUrl(subpath)
	}

	return "", errors.New("Unknown hostname provided in plugin: " + p.ref.Host)
}

func (p *gitPlugin) gitlabUrl(subpath string) (string, error) {
	file, err := url.JoinPath(p.ref.Dir, subpath)

	if err != nil {
		return "", err
	}

	repoPath, err := url.JoinPath(p.ref.Owner, p.ref.Repo)

	if err != nil {
		return "", err
	}

	path, err := url.JoinPath(
		"https://gitlab.com/api/v4/projects",
		url.PathEscape(repoPath),
		"repository",
		"files",
		url.PathEscape(file),
		"raw",
	)

	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(path)

	if err != nil {
		return "", err
	}

	// gitlab doesn't redirect master -> main or main -> master, so using "main"
	// as the default in this case
	query := parsed.Query()
	query.Add("ref", cmp.Or(p.ref.Rev, p.ref.Ref, "main"))
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (p *gitPlugin) request(contentURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, contentURL, nil)

	if err != nil {
		return nil, err
	}

	// Add github token to request if available
	ghToken := os.Getenv("GITHUB_TOKEN")

	if ghToken != "" {
		authValue := fmt.Sprintf("token %s", ghToken)
		req.Header.Add("Authorization", authValue)
	}

	return req, nil
}

func (p *gitPlugin) LockfileKey() string {
	return p.ref.String()
}
