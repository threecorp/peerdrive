package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	"golang.org/x/xerrors"

	"github.com/threecorp/peerdrive/pkg/p2p"
	"github.com/threecorp/peerdrive/pkg/snap"
)

type args struct {
	Rendezvous string
	Port       int
	SyncDir    string
}

func parseArgs() (*args, error) {
	a := &args{}

	flag.StringVar(&a.Rendezvous, "rv", "", "Rendezvous string like the only master key")
	flag.IntVar(&a.Port, "port", 6868, "vpn-mesh port")
	flag.StringVar(&a.SyncDir, "sdir", "./", "Synchornize directory")

	flag.Parse()

	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, r := range []string{"rv"} {
		if !seen[r] {
			return nil, xerrors.Errorf("missing required -%s argument/flag", r)
		}
	}
	syncDir, err := filepath.Abs(a.SyncDir)
	if err != nil {
		return nil, xerrors.Errorf("required -sdir argument/flag: %w", err)
	}
	a.SyncDir = syncDir

	return a, nil
}

func main() {
	// Arguments
	args, err := parseArgs()
	if err != nil {
		log.Fatalf("parseArgs: %+v\n", err)
	}
	ctx := context.Background()

	// P2P Host
	node, err := p2p.NewNode(ctx, args.Port, args.Rendezvous)
	if err != nil {
		log.Fatalf("newNode: %+v\n", err)
	}
	defer node.Close()
	log.Printf("Peer: %s\n", node.Host.ID())

	// Packet
	node.Host.SetStreamHandler(snap.Protocol, snap.RWHandler(node))

	// Synchornize
	go snap.SnapWatcher(node, args.SyncDir)

	// Event Watcher
	snap.SyncWatcher(node, args.SyncDir)
}
