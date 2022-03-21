package hetzner

import (
	"context"

	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

type DNSProvider struct {
	Client *hcloud.Client
}

func NewDNSProvider(cfg *ertia.Project) *DNSProvider {
	return &DNSProvider{
		Client: hcloud.NewClient(hcloud.WithToken(cfg.ProviderToken)),
	}
}

func (p *DNSProvider) Name() string {
	return "hetzner"
}

func (p *DNSProvider) CreateRecord(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	return cfg, nil
}
