package plugin

func Update() error {
	return githubCache.Clear()
}
