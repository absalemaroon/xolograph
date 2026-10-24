package chain

import (
	"context"
	"sort"
)

// Provider is the data source behind one network. All adapters (EVM, Solana, Bitcoin) satisfy it.
type Provider interface {
	Address(ctx context.Context, addr string) (*AddressSummary, error)
	Transaction(ctx context.Context, hash string) (*Transaction, error)
	Activity(ctx context.Context, addr, kind, cursor string) (*ActivityPage, error)
}

// Registry maps network ids to providers.
type Registry struct {
	m    map[string]Provider
	info map[string]NetworkInfo
}

func NewRegistry() *Registry {
	return &Registry{m: map[string]Provider{}, info: map[string]NetworkInfo{}}
}

func (r *Registry) Add(info NetworkInfo, p Provider) {
	r.m[info.ID] = p
	r.info[info.ID] = info
}

func (r *Registry) Get(id string) (Provider, NetworkInfo, bool) {
	p, ok := r.m[id]
	return p, r.info[id], ok
}

// Networks lists registered networks, Ethereum first and the rest alphabetically by name.
func (r *Registry) Networks() []NetworkInfo {
	out := make([]NetworkInfo, 0, len(r.info))
	for _, i := range r.info {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool {
		if (out[a].ID == "ethereum-mainnet") != (out[b].ID == "ethereum-mainnet") {
			return out[a].ID == "ethereum-mainnet"
		}
		return out[a].Name < out[b].Name
	})
	return out
}
