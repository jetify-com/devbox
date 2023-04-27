package lock

type devboxProject interface {
	ConfigHash() (string, error)
	NixPkgsCommitHash() string
	ProjectDir() string
}

type resolver interface {
	IsVersionedPackage(pkg string) bool
	Resolve(pkg, version string) (*Package, error)
}

type Locker interface {
	devboxProject
	Resolve(pkg string) (string, error)
}
