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
	plugin.name = githubNameRegexp.ReplaceAllString(
		strings.Join(lo.Compact([]string{ref.Owner, ref.Repo, name}), "."),
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

func (p *gitPlugin) FileContent(subpath string) ([]byte, error) {
	contentURL, err := p.url(subpath)

	if err != nil {
		return nil, err
	}

	readFile := func() ([]byte, time.Duration, error) {
		archive := filepath.Join("/", "tmp", p.ref.Dir+".tar.gz")
		args := strings.Fields(contentURL)
		cmd := exec.Command(args[0], args[1:]...) // Maybe make async?

		_, err := cmd.Output()

		if err != nil {
			slog.Error("Error executing git archive: ", err)
			return nil, 24 * time.Hour, err
		}

		reader, err := os.Open(archive)
		io.ReadAll(reader)
		err = fileutil.Untar(reader, "/tmp") // TODO: add UUID?

		file, err := os.Open(contentURL)
		info, err := file.Stat()

		if err != nil || info.Size() == 0 {
			return nil, 0, err
		}

		defer file.Close()
		body, err := io.ReadAll(file)

		if err != nil {
			return nil, 0, err
		}

		return body, 24 * time.Hour, nil
	}

	retrieve := func() ([]byte, time.Duration, error) {
		req, err := p.request(contentURL)

		if err != nil {
			return nil, 0, err
		}

		client := &http.Client{}
		res, err := client.Do(req)

		if err != nil {
			return nil, 0, err
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, 0, usererr.New(
				"failed to get plugin %s @ %s (Status code %d). \nPlease make "+
					"sure a plugin.json file exists in plugin directory.",
				p.LockfileKey(),
				req.URL.String(),
				res.StatusCode,
			)
		}

		body, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, 0, err
		}

		// Cache for 24 hours. Once we store the plugin in the lockfile, we
		// should cache this indefinitely and only invalidate if the plugin
		// is updated.
		return body, 24 * time.Hour, nil
	}

	switch p.ref.Type {
	case flake.TypeSSH:
		return sshCache.GetOrSet(contentURL, readFile)
	case flake.TypeGitHub:
		return githubCache.GetOrSet(contentURL, retrieve)
	case flake.TypeGitLab:
		return gitlabCache.GetOrSet(contentURL, retrieve)
	case flake.TypeBitBucket:
		return bitbucketCache.GetOrSet(contentURL, retrieve)
	default:
		return nil, err
	}
}

func (p *gitPlugin) url(subpath string) (string, error) {
	switch p.ref.Type {
	case flake.TypeSSH:
		return p.sshGitUrl()
	case flake.TypeGitLab:
		return p.gitlabUrl(subpath)
	case flake.TypeGitHub:
		return p.githubUrl(subpath)
	case flake.TypeBitBucket:
		return p.bitbucketUrl(subpath)
	default:
		return "", nil
	}
}

func (p *gitPlugin) sshGitUrl() (string, error) {
	address, err := url.Parse(p.ref.URL)

	if err != nil {
		return "", err
	}

	defaultBranch := "main"

	if address.Host == flake.TypeGitHub {
		// using master for GitHub repos for the same reasoning established in `githubUrl`
		defaultBranch = "master"
	}

	fileFormat := "tar.gz"
	baseCommand := fmt.Sprintf("git archive --format=%s --remote=ssh://git@", fileFormat)

	path, _ := url.JoinPath(p.ref.Owner, p.ref.Subgroup, p.ref.Repo)

	archive := filepath.Join("/", "tmp", p.ref.Dir+"."+fileFormat)
	branch := cmp.Or(p.ref.Rev, p.ref.Ref, defaultBranch)

	host := p.ref.Host

	if p.ref.Port != "" {
		host += ":" + p.ref.Port
	}

	// TODO: try to use the Devbox file hashing mechanism to make sure it's stored properly
	command := fmt.Sprintf("%s%s/%s %s %s -o %s", baseCommand, host, path, branch, p.ref.Dir, archive)

	slog.Debug("Generated git archive command: " + command)

	return command, nil

	//sshCache.GetOrSet(command, func() ([]byte, time.Duration, error) {

	//})

	// 24 hours is currently when files are considered "expired" in other FileContent function
	//currentTime := time.Now()
	//threshold := 24 * time.Hour
	//expiration := currentTime.Add(-threshold)

	//args := strings.Fields(command)
	//archiveInfo, err := os.Stat(archive)

	//if os.IsNotExist(err) || archiveInfo.ModTime().Before(expiration) {
	//	cmd := exec.Command(args[0], args[1:]...) // Maybe make async?

	//	_, err := cmd.Output()

	//	if err != nil {
	//		slog.Error("Error executing git archive: ", err)
	//		return "", err
	//	}

	//	reader, err := os.Open(archive)
	//	io.ReadAll(reader)
	//	err = fileutil.Untar(reader, "/tmp") // TODO: add UUID?

	//	if err == nil {
	//		return "", err
	//	}
	//}

	//return filepath.Join("/", "tmp", p.ref.Dir, "plugin.json"), nil
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

func (p *gitPlugin) gitlabUrl(subpath string) (string, error) {
	project, err := url.JoinPath(p.ref.Owner, p.ref.Subgroup, p.ref.Repo)

	if err != nil {
		return "", err
	}

	file, err := url.JoinPath(p.ref.Dir, subpath)

	if err != nil {
		return "", err
	}

	path, err := url.JoinPath(
		"https://gitlab.com/api/v4/projects",
		url.PathEscape(project),
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
