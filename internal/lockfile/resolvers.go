package lockfile

type devboxProject interface {
	ConfigHash() (string, error)
	ProjectDir() string
}

type resolver interface {
	IsVersionedPackage(pkg string) bool
	Resolve(pkg, version string) (*PackageLock, error)
}
