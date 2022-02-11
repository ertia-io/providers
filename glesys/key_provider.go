package glesys

import (
	"context"
	cfg "github.com/ertia-io/config/pkg/entities"
	"github.com/glesys/glesys-go/v3"
)

type GlesysKeyProvider struct{
	Client *glesys.Client
}

func NewKeyProvider(cfg *cfg.Project) *GlesysKeyProvider {
	return &GlesysKeyProvider{
		Client: glesys.NewClient(cfg.ProviderID, cfg.ProviderToken,ErtiaUserAgent),
	}
}

func(p *GlesysKeyProvider) Name() string{
	return "glesys"
}


func(p *GlesysKeyProvider) CreateKey(ctx context.Context, cfg *cfg.Project, key *cfg.SSHKey) (*cfg.Project, error) {
	return cfg,nil
}

func(p *GlesysKeyProvider) DeleteKey(ctx context.Context, cfg *cfg.Project) (*cfg.Project, error) {
	return cfg,nil
}

func(p *GlesysKeyProvider) SyncKeys(ctx context.Context, cfg *cfg.Project) (*cfg.Project, error) {
	return cfg,nil
}