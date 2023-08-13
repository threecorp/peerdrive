package main

import (
	"context"
	"flag"
	"log"

	"golang.org/x/xerrors"

	"github.com/threecorp/peerdrive/pkg/p2p"
	"github.com/threecorp/peerdrive/pkg/sync"
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
			return nil, xerrors.Errorf("missing required -%s argument/flag\n", r)
		}
	}

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

	// Packet
	node.Host.SetStreamHandler(sync.SyncProtocol, sync.SyncHandler(node))

	// Synchornize
	sync.SyncWatcher(node, args.SyncDir)
}
