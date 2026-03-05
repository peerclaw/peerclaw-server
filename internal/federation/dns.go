package federation

import (
	"fmt"
	"net"
)

// DiscoverPeers discovers federated peers via DNS SRV records.
// It looks up _peerclaw._tcp.<domain> SRV records.
func DiscoverPeers(domain string) ([]FederationPeer, error) {
	_, addrs, err := net.LookupSRV("peerclaw", "tcp", domain)
	if err != nil {
		return nil, fmt.Errorf("DNS SRV lookup for %s: %w", domain, err)
	}

	peers := make([]FederationPeer, 0, len(addrs))
	for _, addr := range addrs {
		host := addr.Target
		// Remove trailing dot from DNS name.
		if len(host) > 0 && host[len(host)-1] == '.' {
			host = host[:len(host)-1]
		}
		peers = append(peers, FederationPeer{
			Name:    host,
			Address: fmt.Sprintf("http://%s:%d", host, addr.Port),
		})
	}
	return peers, nil
}
