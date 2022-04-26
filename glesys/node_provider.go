package glesys

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/ertia-io/providers/dependencies"
	"github.com/ertia-io/providers/k3s"
	"github.com/glesys/glesys-go/v3"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const ErtiaUserAgent = "ERTIA: Frictionless Kubernetes"

var DefaultGlesysNode = glesys.CreateServerParams{
	Bandwidth:    100,
	CampaignCode: "",
	CPU:          8,
	DataCenter:   "Falkenberg",
	Memory:       12288,
	Platform:     "KVM",
	Storage:      150,
	IPv4:         "any",
	IPv6:         "any",
	Template:     "debian-11",
}

type GlesysNodeProvider struct {
	Client *glesys.Client
}

func NewNodeProvider(cfg *ertia.Project) *GlesysNodeProvider {
	return &GlesysNodeProvider{
		Client: glesys.NewClient(cfg.ProviderID, cfg.ProviderToken, ErtiaUserAgent),
	}
}

func (p *GlesysNodeProvider) Name() string {
	return "glesys"
}

func (p *GlesysNodeProvider) CreateNode(ctx context.Context, cfg *ertia.Project, node *ertia.Node) (*ertia.Project, error) {
	defaultNode := DefaultGlesysNode

	defaultNode.PublicKey = cfg.SSHKey.PublicKey
	defaultNode.Hostname = node.Name

	defaultNode.Password = string(uuid.NewUUID())

	defaultNode.Users = []glesys.User{{
		Username: "ertia",
		PublicKeys: []string{
			defaultNode.PublicKey,
		},
		Password: defaultNode.Password,
	}}

	node.InstallPassword = defaultNode.Password
	node.InstallUser = "ertia"

	result, err := p.Client.Servers.Create(ctx, defaultNode)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		node.Status = ertia.NodeStatusFailing
		node.Error = err.Error()
		cfg = cfg.UpdateNode(node)
		return cfg, err
	}
	node.ProviderID = result.ID

	var foundIPV4 = false
	var foundIPV6 = false
	for _, ipItem := range result.IPList {
		if foundIPV4 && foundIPV6 {
			break
		}

		if !foundIPV6 {
			if ipItem.Version == 6 {
				foundIPV6 = true
				node.IPV6 = net.ParseIP(ipItem.Address)
			}
			if ipItem.Version == 4 {
				foundIPV4 = true
				node.IPV4 = net.ParseIP(ipItem.Address)
			}
		}

	}

	node.Status = ertia.NodeStatusActive

	//Deploy K3S Next
	node.Dependencies = append(node.Dependencies, dependencies.K3SDependency)

	return cfg.UpdateNode(node), nil
}

func (p *GlesysNodeProvider) DeleteNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error) {
	node := cfg.FindNodeByID(nodeId)
	err := p.Client.Servers.Destroy(ctx, node.ProviderID, glesys.DestroyServerParams{KeepIP: false})

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	node.Status = ertia.NodeStatusDeleted
	return cfg.UpdateNode(node), nil
}

func (p *GlesysNodeProvider) RestartNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error) {
	node := cfg.FindNodeByID(nodeId)

	originalStatus := node.Status

	node.Status = ertia.NodeStatusRestarting
	cfg = cfg.UpdateNode(node)

	cfg, err := p.StopNode(ctx, cfg, nodeId)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}
	cfg, err = p.StartNode(ctx, cfg, nodeId)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	node.Status = originalStatus
	return cfg.UpdateNode(node), nil

}

func (p *GlesysNodeProvider) StopNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error) {

	node := cfg.FindNodeByID(nodeId)

	err := p.Client.Servers.Stop(ctx, node.ProviderID, glesys.StopServerParams{})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	node.Status = ertia.NodeStatusStopped
	return cfg.UpdateNode(node), nil

}

func (p *GlesysNodeProvider) StartNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error) {
	node := cfg.FindNodeByID(nodeId)

	err := p.Client.Servers.Start(ctx, node.ProviderID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		return cfg, err
	}

	node.Status = ertia.NodeStatusActive
	return cfg.UpdateNode(node), nil
}

func (p *GlesysNodeProvider) ReplaceNode(ctx context.Context, cfg *ertia.Project, nodeId string) (*ertia.Project, error) {
	node := cfg.FindNodeByID(nodeId)
	return p.CreateNode(ctx, cfg, node)
}

func (p *GlesysNodeProvider) SyncNodes(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	var err error
	for mi := range cfg.Nodes {
		switch cfg.Nodes[mi].Status {
		case ertia.NodeStatusNew:
			cfg, err = p.CreateNode(ctx, cfg, &cfg.Nodes[mi])
			if err != nil {
				//TODO Set key to failing and do NOT continue
				log.Ctx(ctx).Err(err)
				return cfg, err
			}
		}
	}

	return cfg, nil
}

func (p *GlesysNodeProvider) SyncDependencies(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {

	var err error

	for {
		allDone := true
		for i := range cfg.Nodes {
			if cfg.Nodes[i].Requires(dependencies.K3SDependency.Name) {
				fmt.Printf("Node %s requires %s \n", cfg.Nodes[i].Name, dependencies.K3SDependency.Name)
				allDone = false
				if cfg.Nodes[i].IsMaster {
					cfg, err = installK3SMaster(ctx, cfg, &cfg.Nodes[i])
					if err != nil {
						if errors.Is(err, k3s.ErrorSSHNotReady) {
							err = nil
							time.Sleep(1 * time.Second)
							break
						}
						return cfg, err
					}
				} else {
					if cfg.Nodes[i].MasterIP != nil && cfg.Nodes[i].NodeToken != "" {
						err := k3s.InstallK3SAgent(ctx, cfg.Nodes[i], cfg.Nodes[i].MasterIP.String(), cfg.K3SChannel)
						if err != nil {
							if errors.Is(err, k3s.ErrorSSHNotReady) {
								err = nil
								time.Sleep(1 * time.Second)
								break
							}
							return cfg, err
						}

						for di := range cfg.Nodes[i].Dependencies {
							if cfg.Nodes[i].Dependencies[di].Name == dependencies.K3SDependency.Name {
								cfg.Nodes[i].Dependencies[di].Status = ertia.DependencyStatusReady
							}
						}

						cfg = cfg.UpdateNode(&cfg.Nodes[i])

					} else {
						masterNode := cfg.FindMasterNode()
						if masterNode.Fulfils(dependencies.K3SDependency.Name) {
							cfg.Nodes[i].MasterIP = masterNode.IPV4
							cfg.Nodes[i].NodeToken = masterNode.NodeToken
							err := k3s.InstallK3SAgent(ctx, cfg.Nodes[i], masterNode.IPV4.String(), cfg.K3SChannel)
							if err != nil {
								if errors.Is(err, k3s.ErrorSSHNotReady) {
									err = nil
									time.Sleep(1 * time.Second)
									break
								}
								return cfg, err
							}

							for di := range cfg.Nodes[i].Dependencies {
								if cfg.Nodes[i].Dependencies[di].Name == dependencies.K3SDependency.Name {
									cfg.Nodes[i].Dependencies[di].Status = ertia.DependencyStatusReady
								}
							}

							cfg = cfg.UpdateNode(&cfg.Nodes[i])

						} else {
							//TODO: Maybe keep track of dependency status?
						}
					}

				}
			}
		}
		if allDone {
			fmt.Println("All dependencies handled")
			break
		}
	}

	return cfg, nil
}

func boolAddr(b bool) *bool {
	return &b
}

func installK3SMaster(ctx context.Context, cfg *ertia.Project, node *ertia.Node) (*ertia.Project, error) {
	nodeToken, err := k3s.InstallK3SServer(ctx, node.IPV4, node.InstallUser, node.InstallPassword, cfg.K3SChannel)
	if err != nil {
		return cfg, err
	}

	for di := range node.Dependencies {
		if node.Dependencies[di].Name == dependencies.K3SDependency.Name {
			node.Dependencies[di].Status = ertia.DependencyStatusReady
		}
	}

	node.NodeToken = nodeToken
	return cfg.UpdateNode(node), nil
}
