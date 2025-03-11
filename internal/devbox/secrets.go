package devbox

import (
	"context"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/envsec/pkg/envsec"
	"go.jetify.com/envsec/pkg/stores/jetstore"
	"go.jetify.com/pkg/envvar"
)

func (d *Devbox) UninitializedSecrets(ctx context.Context) *envsec.Envsec {
	return &envsec.Envsec{
		APIHost: build.JetpackAPIHost(),
		Auth: envsec.AuthConfig{
			ClientID: envvar.Get("ENVSEC_CLIENT_ID", build.ClientID()),
			Issuer:   envvar.Get("ENVSEC_ISSUER", build.Issuer()),
		},
		IsDev:      build.IsDev,
		Stderr:     d.stderr,
		Store:      &jetstore.JetpackAPIStore{},
		WorkingDir: d.ProjectDir(),
	}
}

func (d *Devbox) Secrets(ctx context.Context) (*envsec.Envsec, error) {
	envsecInstance := d.UninitializedSecrets(ctx)

	project, err := envsecInstance.ProjectConfig()
	if err != nil {
		return nil, err
	}

	envsecInstance.EnvID = envsec.EnvID{
		EnvName:   d.environment,
		OrgID:     project.OrgID.String(),
		ProjectID: project.ProjectID.String(),
	}

	if _, err := envsecInstance.InitForUser(ctx); err != nil {
		return nil, err
	}

	return envsecInstance, nil
}
