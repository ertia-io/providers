package glesys

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"time"

	ertia "github.com/ertia-io/config/pkg/entities"
	"github.com/glesys/glesys-go/v3"
)

const (
	halfHour = 1800
	IPv4     = "A"
)

type DNSProvider struct {
	Client *glesys.Client
}

func NewDNSProvider(cfg *ertia.Project) *DNSProvider {
	return &DNSProvider{
		Client: glesys.NewClient(cfg.ProviderID, cfg.ProviderToken, ErtiaUserAgent),
	}
}

func (p *DNSProvider) Name() string {
	return "glesys"
}

func (p *DNSProvider) CreateRecord(ctx context.Context, cfg *ertia.Project) (*ertia.Project, error) {
	if !cfg.DNS.NeedsAdapting() {
		return cfg, nil
	}

	ip, err := getDomainIP(cfg)
	if err != nil {
		return cfg, err
	}

	dns := cfg.DNS
	if dns == nil {
		return cfg, fmt.Errorf("DNS configuration not found")
	}

	domainSufix := dns.Domain
	if len(domainSufix) == 0 {
		return cfg, fmt.Errorf("empty domain found")
	}
	host := fmt.Sprintf("*%s", domainSufix)

	domain, err := getDomain(domainSufix)
	if err != nil {
		return cfg, err
	}

	if _, err := p.Client.DNSDomains.Details(ctx, domain); err != nil {
		return cfg, err
	}

	recordID, ok, err := p.findDNSRecord(ctx, domain, host)
	if err != nil {
		return cfg, err
	}

	if !ok {
		newRecord := glesys.AddRecordParams{
			DomainName: domain,
			Host:       host,
			Data:       ip.String(),
			TTL:        halfHour,
			Type:       IPv4,
		}

		if _, err := p.Client.DNSDomains.AddRecord(ctx, newRecord); err != nil {
			return cfg, err
		}
	} else {
		updateRecord := glesys.UpdateRecordParams{
			RecordID: recordID,
			Data:     ip.String(),
		}

		if _, err := p.Client.DNSDomains.UpdateRecord(ctx, updateRecord); err != nil {
			return cfg, err
		}
	}

	dns.Status = ertia.DNSStatusReady
	dns.IPV4 = ip
	dns.Updated = time.Now()

	return cfg.UpdateDNS(dns), nil
}

func (p *DNSProvider) findDNSRecord(ctx context.Context, domain, host string) (int, bool, error) {
	domainRecords, err := p.Client.DNSDomains.ListRecords(ctx, domain)
	if err != nil {
		return 0, false, err
	}

	for _, dr := range *domainRecords {
		if dr.Host == host {
			return dr.RecordID, true, nil
		}
	}

	return 0, false, nil
}

func getDomain(fqdn string) (string, error) {
	re := regexp.MustCompile(`.*\.([a-z0-9-]*\.[a-z]*)`)
	domain := re.FindStringSubmatch(fqdn)
	if domain == nil {
		return "", fmt.Errorf("could not find domain in: %s", fqdn)
	}

	return domain[1], nil
}

func getDomainIP(cfg *ertia.Project) (net.IP, error) {
	node := cfg.FindNonMasterNode()

	if node.Status != ertia.NodeStatusActive {
		return nil, fmt.Errorf("non master node: %s, is not active", node.Name)
	}

	if node.IPV4 == nil {
		return nil, fmt.Errorf("no IPv4 found in node: %s", node.Name)
	}

	return node.IPV4, nil
}
