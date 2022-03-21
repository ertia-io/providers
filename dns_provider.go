package providers

import (
	"context"

	ertia "github.com/ertia-io/config/pkg/entities"
)

type DNSProvider interface {
	Name() string
	CreateRecord(context.Context, *ertia.Project) (*ertia.Project, error)
}
