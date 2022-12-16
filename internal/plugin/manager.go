package plugin

type Manager struct {
	addMode bool
}

type managerOption func(*Manager)

func NewManager(opts ...managerOption) *Manager {
	m := &Manager{}
	m.ApplyOptions(opts...)
	return m
}

func WithAddMode() managerOption {
	return func(m *Manager) {
		m.addMode = true
	}
}

func (m *Manager) ApplyOptions(opts ...managerOption) {
	for _, opt := range opts {
		opt(m)
	}
}
