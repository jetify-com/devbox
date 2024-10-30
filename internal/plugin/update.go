package plugin

import "go.jetpack.io/pkg/filecache"

func Update() error {
	pluginCaches := []*filecache.Cache[[]byte]{githubCache, sshCache, gitlabCache, bitbucketCache}

	for _, cache := range pluginCaches {
		err := cache.Clear()

		if err != nil {
			return err
		}

	}

	return nil
}
