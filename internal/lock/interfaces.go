package lock

type devboxProject interface {
	ConfigHash() (string, error)
	IsPlugin(name string) bool
	ProjectDir() string
}

type resolver interface {
	IsVersionedPackage(pkg string) bool
	Resolve(pkg, version string) (*Package, error)
}

type Locker interface {
	devboxProject
	IsVersionedPackage(pkg string) bool
	Resolve(pkg string) (string, error)
}
