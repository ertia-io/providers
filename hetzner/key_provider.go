package hetzner

import (
	"context"
	"fmt"
	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/rs/zerolog/log"
	"strconv"
)

type HetznerKeyProvider struct {
	Client *hcloud.Client
}

func NewKeyProvider(cfg *ertia.Project) *HetznerKeyProvider {
	return &HetznerKeyProvider{
		Client: hcloud.NewClient(hcloud.WithToken(cfg.ProviderToken)),
	}
}

func (p *HetznerKeyProvider) Name() string {
	return "hetzner"
}

func (p *HetznerKeyProvider) CreateKey(ctx context.Context, cfg *ertia.Project, key *ertia.SSHKey) (*ertia.Project, error) {

	key.Status = ertia.KeyStatusAdapting

	cfg = cfg.UpdateKey(key)

	//Create a key in hetzner.
	result, _, err := p.Client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      key.Name,
		PublicKey: key.PublicKey,
	})

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		key.Status = ertia.KeyStatusFailing
		key.Error = err.Error()
		c := cfg.UpdateKey(key)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Send()
		}
		return c, err
	}

	key.ProviderID = fmt.Sprintf("%d", result.ID)
	key.Status = ertia.KeyStatusActive
	key.Fingerprint = result.Fingerprint

	cfg = cfg.UpdateKey(key)

	return cfg, err
}

func (p *HetznerKeyProvider) DeleteKey(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	key := cfg.SSHKey
	pid, err := strconv.Atoi(key.ProviderID)
	if err != nil {
		return cfg, err
	}
	_, err = p.Client.SSHKey.Delete(ctx, &hcloud.SSHKey{
		ID: pid,
	})
	if err != nil {
		return cfg, err
	}

	key.Status = ertia.KeyStatusDeleted
	return cfg.UpdateKey(key), nil
}

func (p *HetznerKeyProvider) SyncKeys(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	var err error
	switch cfg.SSHKey.Status {
	case ertia.KeyStatusNew:
		cfg, err = p.CreateKey(ctx, cfg, cfg.SSHKey)
		if err != nil {
			panic("COuld not create key:" + err.Error())
			//TODO Set key to failing and do NOT continue
			log.Ctx(ctx).Err(err)
			return cfg, err
		}
	}

	return cfg, nil
}
