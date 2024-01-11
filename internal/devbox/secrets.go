package devbox

import (
	"context"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/envsec/pkg/envsec"
	"go.jetpack.io/envsec/pkg/stores/jetstore"
	"go.jetpack.io/pkg/envvar"
)

type secrets struct {
	envsec.Envsec
	EnvName string
}

func (d *Devbox) Secrets(ctx context.Context) (*secrets, error) {
	envsecInstance := envsec.Envsec{
		APIHost: build.JetpackAPIHost(),
		Auth: envsec.AuthConfig{
			ClientID: envvar.Get("ENVSEC_CLIENT_ID", build.ClientID()),
			Issuer:   envvar.Get("ENVSEC_ISSUER", build.Issuer()),
		},
		IsDev:      build.IsDev,
		Stderr:     d.stderr,
		WorkingDir: d.ProjectDir(),
	}

	store := &jetstore.JetpackAPIStore{}
	if err := envsecInstance.SetStore(ctx, store); err != nil {
		return nil, err
	}
	return &secrets{
		Envsec:  envsecInstance,
		EnvName: d.environment,
	}, nil
}

func (s *secrets) EnvID() (envsec.EnvID, error) {
	project, err := s.ProjectConfig()
	if err != nil {
		return envsec.EnvID{}, err
	}
	return envsec.EnvID{
		EnvName:   s.EnvName,
		ProjectID: project.ProjectID.String(),
		OrgID:     project.OrgID.String(),
	}, nil
}
