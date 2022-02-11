package hetzner

import (
	"context"
	"errors"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/rs/zerolog/log"
	"lube/pkg/dependencies"
	"lube/pkg/k3s"
	"lube/pkg/lubeconfig/entities"
	"strconv"
	"time"
)

type HetznerNodeProvider struct{

}

func NewNodeProvider() *HetznerNodeProvider {
	return &HetznerNodeProvider{}
}

func(p *HetznerNodeProvider) Name() string{
	return "hetzner"
}


func(p *HetznerNodeProvider) CreateNode(ctx context.Context, cfg *entities.LubeConfig, node *entities.LubeNode) (*entities.LubeConfig, error){

	hc := hcloud.NewClient(hcloud.WithToken(cfg.APIToken))


	sshKeys := []*hcloud.SSHKey{}


	for i := range cfg.SSHKeys{

		intId, err := strconv.Atoi(cfg.SSHKeys[i].ProviderID)
		if(err!=nil){
			continue
		}

		sshKeys = append(sshKeys, &hcloud.SSHKey{
			ID:        intId,
		})
	}

	//Create a kvm in hetzner.
	result, _, err := hc.Server.Create(context.Background(), hcloud.ServerCreateOpts{
		Name: node.Name,
		ServerType: &hcloud.ServerType{
			ID: 1, //TODO: COnfigurable. 3= CX21?
		},
		Image: &hcloud.Image{
			Name: "ubuntu-20.04", //TODO Check ID?
		},
		SSHKeys:          sshKeys,
		Location:         nil, // TODO: Make this selectable
		Datacenter:       nil, // TODO: Make this selectable
		StartAfterCreate: boolAddr(true),
		Labels:           nil, // TODO: Add tags to machine/cluster and apply these here..?
		Automount:        nil,
		Volumes:          nil,
		Networks:         nil,
		Firewalls:        nil,
		PlacementGroup:   nil,
	})


	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		node.Status = entities.NodeStatusFailing
		node.Error = err.Error()
		c, err := cfg.UpdateNode(node)
		if (err != nil) {
			log.Ctx(ctx).Error().Err(err).Send()
		}
		cfg = c
		return c, err
	}
	node.ProviderID = fmt.Sprintf("%d", result.Server.ID)
	node.IPV4 = result.Server.PublicNet.IPv4.IP
	node.IPV6 = result.Server.PublicNet.IPv6.IP
	node.Status = entities.NodeStatusActive
	node.InstallUser = "root"

	//Deploy K3S Next
	node.Dependencies = append(node.Dependencies, dependencies.K3SDependency)

	return cfg.UpdateNode(node)
}

func(p *HetznerNodeProvider)  DeleteNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){

	hc := hcloud.NewClient(hcloud.WithToken(cfg.APIToken))

	node := cfg.FindNodeByID(nodeId)

	providerId, err := strconv.Atoi(node.ProviderID)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}
	_, err = hc.Server.Delete(ctx, &hcloud.Server{ID: providerId})
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	node.Status = entities.NodeStatusDeleted
	return cfg.UpdateNode(node)
}


func(p *HetznerNodeProvider)  RestartNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	hc := hcloud.NewClient(hcloud.WithToken(cfg.APIToken))

	node := cfg.FindNodeByID(nodeId)
	providerId, err := strconv.Atoi(node.ProviderID)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	originalStatus := node.Status

	node.Status = entities.NodeStatusRestarting
	cfg, err = cfg.UpdateNode(node)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}
	_, _, err = hc.Server.Reboot(ctx, &hcloud.Server{ID: providerId})

	node.Status = originalStatus
	cfg, err = cfg.UpdateNode(node)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	return cfg, nil
}

func(p *HetznerNodeProvider)  StopNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	hc := hcloud.NewClient(hcloud.WithToken(cfg.APIToken))

	node := cfg.FindNodeByID(nodeId)
	providerId, err := strconv.Atoi(node.ProviderID)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	_, _, err = hc.Server.Shutdown(ctx, &hcloud.Server{ID: providerId})

	node.Status = entities.NodeStatusStopped
	cfg, err = cfg.UpdateNode(node)
	if (err != nil) {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	return cfg, nil
}

func(p *HetznerNodeProvider)  StartNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	return p.RestartNode(ctx, cfg, nodeId)
}

func(p *HetznerNodeProvider)  ReplaceNode(ctx context.Context, cfg *entities.LubeConfig, nodeId string) (*entities.LubeConfig, error){
	node := cfg.FindNodeByID(nodeId)
	return p.CreateNode(ctx, cfg, node)
}


func(p *HetznerNodeProvider) SyncNodes(ctx context.Context, cfg *entities.LubeConfig) (*entities.LubeConfig, error) {
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

func (p *HetznerNodeProvider) SyncDependencies(ctx context.Context, cfg *entities.LubeConfig) (*entities.LubeConfig, error) {

	var err error

	for {
		allDone := true
		for i := range cfg.Nodes {
			if (cfg.Nodes[i].Requires(dependencies.K3SDependency.Name)) {
				fmt.Printf("Node %s requires %s \n",cfg.Nodes[i].Name, dependencies.K3SDependency.Name)
				allDone = false
				if (cfg.Nodes[i].IsMaster) {
					cfg, err = installK3SMaster(ctx, cfg, &cfg.Nodes[i])
					if (err != nil) {
						if(errors.Is(err,k3s.ErrorSSHNotReady)){
							err = nil
							time.Sleep(1*time.Second)
							break
						}
						return cfg, err
					}
				} else {
					if (cfg.Nodes[i].MasterIP != nil && cfg.Nodes[i].NodeToken != "") {
						err := k3s.InstallK3SAgent(ctx, cfg.Nodes[i], cfg.Nodes[i].MasterIP.String())
						if (err != nil) {
							if(errors.Is(err,k3s.ErrorSSHNotReady)){
								err = nil
								time.Sleep(1*time.Second)
								break
							}
							return cfg, err
						}

						for di := range cfg.Nodes[i].Dependencies{
							if(cfg.Nodes[i].Dependencies[di].Name == dependencies.K3SDependency.Name){
								cfg.Nodes[i].Dependencies[di].Status = entities.DependencyStatusReady
							}
						}

						cfg, err = cfg.UpdateNode(&cfg.Nodes[i])
						if(err!=nil){
							return cfg, err
						}

					} else {
						masterNode := cfg.FindClusterMasterNode(cfg.Nodes[i].ClusterName)
						if (masterNode.Fulfils(dependencies.K3SDependency.Name)) {
							cfg.Nodes[i].MasterIP = masterNode.IPV4
							cfg.Nodes[i].NodeToken = masterNode.NodeToken
							err := k3s.InstallK3SAgent(ctx, cfg.Nodes[i], masterNode.IPV4.String())
							if (err != nil) {
								if(errors.Is(err,k3s.ErrorSSHNotReady)){
									err = nil
									time.Sleep(1*time.Second)
									break
								}
								return cfg, err
							}

							for di := range cfg.Nodes[i].Dependencies{
								if(cfg.Nodes[i].Dependencies[di].Name == dependencies.K3SDependency.Name){
									cfg.Nodes[i].Dependencies[di].Status = entities.DependencyStatusReady
								}
							}

							cfg, err = cfg.UpdateNode(&cfg.Nodes[i])
							if(err!=nil){
								return cfg, err
							}

						} else {
							//TODO: Maybe keep track of dependency status?
						}
					}

				}
			}
		}
		if(allDone){
			fmt.Println("All dependencies handled")
			break
		}
	}

	return cfg, nil
}

func boolAddr(b bool) *bool{
	return &b
}

func installK3SMaster(ctx context.Context, cfg *entities.LubeConfig, node *entities.LubeNode) (*entities.LubeConfig, error){
	nodeToken, err := k3s.InstallK3SServer(ctx, node.IPV4,node.InstallUser, node.InstallPassword)
	if(err!=nil){
		return cfg, err
	}

	for di := range node.Dependencies{
		if(node.Dependencies[di].Name == dependencies.K3SDependency.Name){
			node.Dependencies[di].Status = entities.DependencyStatusReady
		}
	}

	node.NodeToken = nodeToken
	cfg, err = cfg.UpdateNode(node)
	if(err!=nil){
		return cfg, err
	}
	return cfg, nil
}