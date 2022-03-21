package providers

import (
	"context"

	ertia "github.com/ertia-io/config/pkg/entities"
)

type NodeProvider interface {
	Name() string
	CreateNode(ctx context.Context, cfg *ertia.Project, node *ertia.Node) (*ertia.Project, error)
	DeleteNode(context.Context, *ertia.Project, string /*nodeId*/) (*ertia.Project, error)
	RestartNode(context.Context, *ertia.Project, string /*nodeId*/) (*ertia.Project, error)
	StopNode(context.Context, *ertia.Project, string /*nodeId*/) (*ertia.Project, error)
	StartNode(context.Context, *ertia.Project, string /*nodeId*/) (*ertia.Project, error)
	ReplaceNode(context.Context, *ertia.Project, string /*nodeId*/) (*ertia.Project, error)
	SyncNodes(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error)
	SyncDependencies(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error)
}
