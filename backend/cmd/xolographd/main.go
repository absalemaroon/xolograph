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
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/xolograph/backend/pkg/api"
	"github.com/xolograph/backend/pkg/store"
)

func main() {
	dataDir := flag.String("data-dir", "", "application data directory (default: user config dir/Xolograph)")
	port := flag.Int("port", 0, "loopback port (0 = random)")
	token := flag.String("token", "", "API token (default: random; use a fixed value only for development)")
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
	srv := &http.Server{Handler: api.New(st, *token).Handler(), ReadHeaderTimeout: 10 * time.Second}

	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"addr": ln.Addr().String(), "token": *token})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
