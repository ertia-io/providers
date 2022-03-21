package k3d

import (
	"context"

	ertia "github.com/ertia-io/config/pkg/entities"
)

type DNSProvider struct{}

func NewDNSProvider() *DNSProvider {
	return &DNSProvider{}
}

func (p *DNSProvider) Name() string {
	return "k3d"
}

func (p *DNSProvider) CreateRecord(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	return cfg, nil
}
