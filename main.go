package main

import (
	"context"
	"flag"
	"log"

	"github.com/pkg/errors"
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
			return nil, errors.Errorf("missing required -%s argument/flag\n", r)
		}
	}

	return a, nil
}

func main() {
	// Arguments
	args, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// P2P Host
	node, err := p2p.NewNodeByLite(ctx, args.Port, args.Rendezvous)
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()

	// Packet
	node.Host.SetStreamHandler(sync.SyncProtocol, sync.SyncHandler(node))

	// Synchornize
	sync.SyncWatcher(node, args.SyncDir)
}
