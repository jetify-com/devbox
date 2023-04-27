package plugin

import "go.jetpack.io/devbox/internal/lock"

type Manager struct {
	lockfile *lock.File
}

type managerOption func(*Manager)

func NewManager(opts ...managerOption) *Manager {
	m := &Manager{}
	m.ApplyOptions(opts...)
	return m
}

func WithLockfile(lockfile *lock.File) managerOption {
	return func(m *Manager) {
		m.lockfile = lockfile
	}
}

func (m *Manager) ApplyOptions(opts ...managerOption) {
	for _, opt := range opts {
		opt(m)
	}
}

func (m *Manager) IsPlugin(name string) bool {
	_, err := parseInclude(name)
	return err == nil
}
