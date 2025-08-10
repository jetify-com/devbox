package plugin

func Update() error {
	return gitCache.Clear()
}
