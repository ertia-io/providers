package hetzner

import (
	"context"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/rs/zerolog/log"
	"lube/pkg/keys"
	"lube/pkg/lubeconfig/entities"
	"strconv"
)

type HetznerKeyProvider struct{
	Client *hcloud.Client
}

func NewKeyProvider(cfg *entities.LubeConfig) *HetznerKeyProvider {
	return &HetznerKeyProvider{
		Client:hcloud.NewClient(hcloud.WithToken(cfg.APIToken)),
	}
}

func(p *HetznerKeyProvider) Name() string{
	return "hetzner"
}


func(p *HetznerKeyProvider) CreateKey(ctx context.Context, cfg *entities.LubeConfig, key *keys.LubeSSHKey) (*entities.LubeConfig, error) {

	key.Status = keys.KeyStatusAdapting

	cfg, _ = cfg.UpdateKey(key)


	//Create a key in hetzner.
	result, _, err := p.Client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      key.Name,
		PublicKey: key.PublicKey,
	})


	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		key.Status = keys.KeyStatusFailing
		key.Error = err.Error()
		c, err := cfg.UpdateKey(key)
		if (err != nil) {
			log.Ctx(ctx).Error().Err(err).Send()
		}
		return c, err
	}

	key.ProviderID = fmt.Sprintf("%d", result.ID)
	key.Status = keys.KeyStatusActive
	key.Fingerprint = result.Fingerprint

	cfg, _ = cfg.UpdateKey(key)

	return cfg, err
}

func(p *HetznerKeyProvider) DeleteKey(ctx context.Context, cfg *entities.LubeConfig, keyId string) (*entities.LubeConfig, error) {
	key := cfg.FindKeyByID(keyId)
	pid, err := strconv.Atoi(key.ProviderID)
	if(err!=nil){
		return cfg, err
	}
	_, err = p.Client.SSHKey.Delete(ctx, &hcloud.SSHKey{
		ID:          pid,
	})
	if(err!=nil){
		return cfg, err
	}

	key.Status = keys.KeyStatusDeleted
	return cfg.UpdateKey(key)
}

func(p *HetznerKeyProvider) SyncKeys(ctx context.Context, cfg *entities.LubeConfig) (*entities.LubeConfig, error) {
	var err error
	for mi := range cfg.SSHKeys {
		switch (cfg.SSHKeys[mi].Status){
		case keys.KeyStatusNew:
			cfg, err = p.CreateKey(ctx, cfg,&cfg.SSHKeys[mi])
			if(err!=nil){
				panic("COuld not create key:"+err.Error())
				//TODO Set key to failing and do NOT continue
				log.Ctx(ctx).Err(err)
				return cfg, err
			}
		}
	}

	return cfg, nil
}