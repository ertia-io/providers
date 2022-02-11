package k3d

import (
	"context"
	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/rs/zerolog/log"
)

type K3DNodeProvider struct{

}

func NewNodeProvider() *K3DNodeProvider {
	return &K3DNodeProvider{}
}

func(p *K3DNodeProvider) Name() string{
	return "k3d"
}

func(p *K3DNodeProvider)  CreateNode(ctx context.Context, cfg *ertia.Project, node *ertia.Node) (*ertia.Project, error){
	node.Status = ertia.NodeStatusActive
	cfg.UpdateNode(node)
	return cfg,nil
}
func(p *K3DNodeProvider)  DeleteNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = ertia.NodeStatusDeleted
	cfg.UpdateNode(node)
	return cfg,nil
}


func(p *K3DNodeProvider)  RestartNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error){
	node := cfg.FindNodeByID(nodeId)
	if(node.Status!=ertia.NodeStatusReady){
		node.Status = ertia.NodeStatusActive
	}
	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  StopNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = ertia.NodeStatusStopped
	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  StartNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project,error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = ertia.NodeStatusActive

	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  ReplaceNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error){
	node := cfg.FindNodeByID(nodeId)
	if(node.Status!=ertia.NodeStatusReady){
		node.Status = ertia.NodeStatusActive
	}

	cfg.UpdateNode(node)
	return cfg,nil
}


func(p *K3DNodeProvider) SyncNodes(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	var err error
	for mi := range cfg.Nodes {
		switch (cfg.Nodes[mi].Status){
		case ertia.NodeStatusNew:
			cfg, err = p.CreateNode(ctx, cfg,&cfg.Nodes[mi])
			if(err!=nil){
				//TODO Set key to failing and do NOT continue
				log.Ctx(ctx).Err(err)
				return cfg, err
			}
		}
	}

	return cfg, nil
}

func(p *K3DNodeProvider) SyncDependencies(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error){
	for i := range cfg.Nodes {
		if(cfg.Nodes[i].Status == ertia.NodeStatusActive){
			cfg.Nodes[i].Status = ertia.NodeStatusReady
			cfg =cfg.UpdateNode(&cfg.Nodes[i])
		}
	}
	return cfg, nil
}