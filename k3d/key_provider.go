package k3d

import (
	"context"
	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/rs/zerolog/log"
)

type K3DKeyProvider struct{}

func NewKeyProvider() *K3DKeyProvider {
	return &K3DKeyProvider{}
}

func (p *K3DKeyProvider) Name() string {
	return "k3d"
}

func (p *K3DKeyProvider) CreateKey(context context.Context, cfg *ertia.Project, key *ertia.SSHKey) (*ertia.Project, error) {
	key.Status = ertia.KeyStatusActive
	cfg.UpdateKey(key)
	return cfg, nil
}

func (p *K3DKeyProvider) DeleteKey(context context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	key := cfg.SSHKey
	key.Status = ertia.KeyStatusDeleted
	cfg.UpdateKey(key)
	return cfg, nil
}

func (p K3DKeyProvider) SyncKeys(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	var err error
	switch cfg.SSHKey.Status {
	case ertia.KeyStatusNew:
		cfg, err = p.CreateKey(ctx, cfg, cfg.SSHKey)
		if err != nil {
			//TODO Set key to failing and do NOT continue
			log.Ctx(ctx).Err(err)
			return cfg, err
		}
	}

	return cfg, nil
}
