package providers

import (
	"context"
	ertia "github.com/ertia-io/config/pkg/entities"
)

type KeyProvider interface {
	CreateKey(context.Context, *ertia.Project, *ertia.SSHKey) (*ertia.Project, error)
	DeleteKey(context.Context, *ertia.Project, string /*keyId*/) (*ertia.Project, error)
	SyncKeys(context.Context, *ertia.Project) (*ertia.Project, error)
}