// xolographd is the Xolograph engine, run as a sidecar by the desktop shell.
// On start it prints one JSON line {"addr":..., "token":...} to stdout.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/xolograph/backend/pkg/api"
	"github.com/xolograph/backend/pkg/chain"
	"github.com/xolograph/backend/pkg/store"
)

func main() {
	dataDir := flag.String("data-dir", "", "application data directory (default: user config dir/Xolograph)")
	port := flag.Int("port", 0, "loopback port (0 = random)")
	token := flag.String("token", "", "API token (default: random; use a fixed value only for development)")
	rpcURL := flag.String("rpc-url", envOr("XOLOGRAPH_RPC_URL", "https://ethereum-rpc.publicnode.com"), "Ethereum JSON-RPC endpoint")
	historyURL := flag.String("history-url", envOr("XOLOGRAPH_HISTORY_URL", "https://eth.blockscout.com"), "Blockscout v2 base URL for address history and labels (empty disables)")
	exitOnStdin := flag.Bool("exit-on-stdin-close", false, "exit when stdin closes (set by the desktop shell so the engine never outlives the app)")
	flag.Parse()

	if *dataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			log.Fatalf("cannot resolve config dir: %v", err)
		}
		*dataDir = filepath.Join(base, "Xolograph")
	}
	if *token == "" {
		b := make([]byte, 24)
		if _, err := rand.Read(b); err != nil {
			log.Fatal(err)
		}
		*token = hex.EncodeToString(b)
	}

	st, err := store.Open(*dataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port)) // loopback only
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: api.NewMulti(st, *token, buildRegistry(*rpcURL, *historyURL)).Handler(), ReadHeaderTimeout: 10 * time.Second}

	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"addr": ln.Addr().String(), "token": *token})

	var stdinClosed chan struct{} // nil (blocks forever) unless requested
	if *exitOnStdin {
		stdinClosed = make(chan struct{})
		go func() {
			_, _ = io.Copy(io.Discard, os.Stdin)
			close(stdinClosed)
		}()
	}
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		select {
		case <-sig:
		case <-stdinClosed:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type evmNet struct{ id, name, symbol, rpc, history string }

// Default public endpoints. They are conveniences, not guarantees: override any of them with
// XOLOGRAPH_RPC_<ID> / XOLOGRAPH_HISTORY_<ID> (ID upper-cased, "-" as "_"), or set XOLOGRAPH_RPC_<ID>=off to disable a network.
var evmNets = []evmNet{
	{"base", "Base", "ETH", "https://base-rpc.publicnode.com", "https://base.blockscout.com"},
	{"optimism", "Optimism", "ETH", "https://optimism-rpc.publicnode.com", "https://optimism.blockscout.com"},
	{"arbitrum", "Arbitrum One", "ETH", "https://arbitrum-one-rpc.publicnode.com", "https://arbitrum.blockscout.com"},
	{"polygon", "Polygon", "POL", "https://polygon-bor-rpc.publicnode.com", "https://polygon.blockscout.com"},
	{"bsc", "BNB Smart Chain", "BNB", "https://bsc-rpc.publicnode.com", ""},
	{"gnosis", "Gnosis", "xDAI", "https://gnosis-rpc.publicnode.com", "https://gnosis.blockscout.com"},
	{"avalanche", "Avalanche C-Chain", "AVAX", "https://avalanche-c-chain-rpc.publicnode.com", ""},
	{"scroll", "Scroll", "ETH", "https://scroll-rpc.publicnode.com", "https://scroll.blockscout.com"},
	{"celo", "Celo", "CELO", "https://forno.celo.org", "https://celo.blockscout.com"},
}

func envID(id string) string { return strings.ToUpper(strings.ReplaceAll(id, "-", "_")) }

// netEnv returns the endpoint for a network, honouring XOLOGRAPH_<kind>_<ID>. ok is false when the network is switched off.
func netEnv(kind, id, def string) (string, bool) {
	v := os.Getenv("XOLOGRAPH_" + kind + "_" + envID(id))
	switch strings.ToLower(v) {
	case "":
		return def, def != ""
	case "off", "none", "disabled":
		return "", false
	}
	return v, true
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func buildRegistry(ethRPC, ethHistory string) *chain.Registry {
	reg := chain.NewRegistry()
	var bs *chain.Blockscout
	if ethHistory != "" {
		bs = chain.NewBlockscout(ethHistory)
	}
	reg.Add(chain.NetworkInfo{ID: "ethereum-mainnet", Name: "Ethereum", Family: "EVM", NativeSymbol: "ETH", NativeDecimals: 18,
		HasHistory: bs != nil, HasLabels: bs != nil, Endpoint: hostOf(ethRPC)}, chain.NewEthereum(chain.NewRPC(ethRPC), bs))

	for _, n := range evmNets {
		rpc, ok := netEnv("RPC", n.id, n.rpc)
		if !ok {
			continue
		}
		var b *chain.Blockscout
		if h, hok := netEnv("HISTORY", n.id, n.history); hok {
			b = chain.NewBlockscout(h)
		}
		reg.Add(chain.NetworkInfo{ID: n.id, Name: n.name, Family: "EVM", NativeSymbol: n.symbol, NativeDecimals: 18,
			HasHistory: b != nil, HasLabels: b != nil, Endpoint: hostOf(rpc)}, chain.NewEVM(n.id, n.symbol, chain.NewRPC(rpc), b))
	}
	if rpc, ok := netEnv("RPC", "solana", "https://api.mainnet-beta.solana.com"); ok {
		reg.Add(chain.NetworkInfo{ID: "solana", Name: "Solana", Family: "SOLANA", NativeSymbol: "SOL", NativeDecimals: 9,
			HasHistory: true, Endpoint: hostOf(rpc)}, chain.NewSolana(chain.NewRPC(rpc)))
	}
	if base, ok := netEnv("RPC", "bitcoin", "https://mempool.space/api"); ok {
		reg.Add(chain.NetworkInfo{ID: "bitcoin", Name: "Bitcoin", Family: "BITCOIN", NativeSymbol: "BTC", NativeDecimals: 8,
			HasHistory: true, Endpoint: hostOf(base)}, chain.NewBitcoin(base))
	}
	return reg
}
