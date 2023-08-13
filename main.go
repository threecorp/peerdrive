package main

import (
	"context"
	"log"

	"github.com/threecorp/peerdrive/p2p"
	"github.com/threecorp/peerdrive/sync"
)

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
