package k3d

import (
	"context"
	"github.com/rs/zerolog/log"
	"lube/pkg/lubeconfig/entities"
)

type K3DNodeProvider struct{

}

func NewNodeProvider() *K3DNodeProvider {
	return &K3DNodeProvider{}
}

func(p *K3DNodeProvider) Name() string{
	return "k3d"
}

func(p *K3DNodeProvider)  CreateNode(ctx context.Context, cfg *entities.LubeConfig, node *entities.LubeNode) (*entities.LubeConfig, error){
	node.Status = entities.NodeStatusActive
	cfg.UpdateNode(node)
	return cfg,nil
}
func(p *K3DNodeProvider)  DeleteNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = entities.NodeStatusDeleted
	cfg.UpdateNode(node)
	return cfg,nil
}


func(p *K3DNodeProvider)  RestartNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	node := cfg.FindNodeByID(nodeId)
	if(node.Status!=entities.NodeStatusReady){
		node.Status = entities.NodeStatusActive
	}
	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  StopNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = entities.NodeStatusStopped
	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  StartNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig,error){
	node := cfg.FindNodeByID(nodeId)
	node.Status = entities.NodeStatusActive

	cfg.UpdateNode(node)
	return cfg,nil
}

func(p *K3DNodeProvider)  ReplaceNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	node := cfg.FindNodeByID(nodeId)
	if(node.Status!=entities.NodeStatusReady){
		node.Status = entities.NodeStatusActive
	}

	cfg.UpdateNode(node)
	return cfg,nil
}


func(p *K3DNodeProvider) SyncNodes(ctx context.Context, cfg *entities.LubeConfig) (*entities.LubeConfig, error) {
	var err error
	for mi := range cfg.Nodes {
		switch (cfg.Nodes[mi].Status){
		case entities.NodeStatusNew:
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

func(p *K3DNodeProvider) SyncDependencies(ctx context.Context, cfg *entities.LubeConfig) (*entities.LubeConfig, error){
	var err error
	for i := range cfg.Nodes {
		if(cfg.Nodes[i].Status == entities.NodeStatusActive){
			cfg.Nodes[i].Status = entities.NodeStatusReady
			cfg, err =cfg.UpdateNode(&cfg.Nodes[i])
			if(err!=nil){
				return cfg, err
			}
		}
	}
	return cfg, nil
}