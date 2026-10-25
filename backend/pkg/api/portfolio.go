package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/xolograph/backend/pkg/portfolio"
)

func (s *Server) needRegistry(w http.ResponseWriter) bool {
	if s.reg == nil {
		writeErr(w, 503, "multi-network data sources are not configured")
		return false
	}
	return true
}

func netParam(r *http.Request) string {
	if n := r.URL.Query().Get("network"); n != "" {
		return n
	}
	return "ethereum-mainnet"
}

func (s *Server) walletPortfolio(w http.ResponseWriter, r *http.Request) {
	if !s.needRegistry(w) {
		return
	}
	v, err := portfolio.BuildWallet(r.Context(), s.reg, s.px, netParam(r), r.URL.Query().Get("address"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (s *Server) walletFlows(w http.ResponseWriter, r *http.Request) {
	if !s.needRegistry(w) {
		return
	}
	pages, _ := strconv.Atoi(r.URL.Query().Get("pages"))
	v, err := portfolio.BuildFlows(r.Context(), s.reg, s.px, netParam(r), r.URL.Query().Get("address"), pages)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

type entityWalletValue struct {
	AddressID  string            `json:"addressId"`
	Network    string            `json:"network"`
	Address    string            `json:"address"`
	Label      string            `json:"label"`
	Status     string            `json:"status"`
	Confidence string            `json:"confidence"`
	Portfolio  *portfolio.Wallet `json:"portfolio,omitempty"`
	Error      string            `json:"error,omitempty"`
}

type entityPortfolioResp struct {
	EntityID string              `json:"entityId"`
	Name     string              `json:"name"`
	AsOf     time.Time           `json:"asOf"`
	Wallets  []entityWalletValue `json:"wallets"`
	// StrongUSD counts only VERIFIED and PUBLICLY_REPORTED wallets; AllUSD counts every non-rejected claim.
	StrongUSD   float64            `json:"strongUsd"`
	AllUSD      float64            `json:"allUsd"`
	ByStatusUSD map[string]float64 `json:"byStatusUsd"`
	Skipped     []string           `json:"skipped"`
	Notes       []string           `json:"notes"`
}

const maxEntityWallets = 12

// entityPortfolio values each claimed wallet separately and reports totals by attribution strength, so a total is never
// presented as "what the entity holds" when it includes wallets that are only suspected.
func (s *Server) entityPortfolio(w http.ResponseWriter, r *http.Request) {
	if !s.needRegistry(w) {
		return
	}
	e, err := s.st.GetEntity(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	out := entityPortfolioResp{EntityID: e.ID, Name: e.Name, AsOf: time.Now().UTC(), Wallets: []entityWalletValue{}, ByStatusUSD: map[string]float64{}, Skipped: []string{}, Notes: []string{}}
	seen := map[string]bool{}
	n := 0
	for _, a := range e.Addresses {
		key := a.Network + "|" + a.Address
		switch {
		case a.Status == "REJECTED":
			out.Skipped = append(out.Skipped, a.Address+" ("+a.Network+"): rejected claim")
			continue
		case seen[key]:
			continue
		case n >= maxEntityWallets:
			out.Skipped = append(out.Skipped, a.Address+" ("+a.Network+"): over the "+strconv.Itoa(maxEntityWallets)+"-wallet limit")
			continue
		}
		seen[key] = true
		n++
		item := entityWalletValue{AddressID: a.ID, Network: a.Network, Address: a.Address, Label: a.Label, Status: a.Status, Confidence: a.Confidence}
		wv, err := portfolio.BuildWallet(ctx, s.reg, s.px, a.Network, a.Address)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Portfolio = wv
			out.AllUSD += wv.PricedValue
			out.ByStatusUSD[a.Status] += wv.PricedValue
			if a.Status == "VERIFIED" || a.Status == "PUBLICLY_REPORTED" {
				out.StrongUSD += wv.PricedValue
			}
		}
		out.Wallets = append(out.Wallets, item)
	}
	out.Notes = append(out.Notes,
		"Each wallet is valued separately. Wallet ownership is a claim with its own status; the all-claims total includes suspected and investigator-attributed wallets and must not be read as confirmed holdings.",
		"Totals include only assets that have a reference price. Unpriced assets are listed per wallet and excluded.")
	writeJSON(w, 200, out)
}


