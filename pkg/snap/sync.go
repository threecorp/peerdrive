package snap

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/samber/lo"
	"golang.org/x/xerrors"

	"github.com/threecorp/peerdrive/pkg/event"
	"github.com/threecorp/peerdrive/pkg/p2p"
)

const Protocol = "/peerdrive/snap/1.0.0"

func Handler(nd *p2p.Node) func(stream network.Stream) {
	return func(stream network.Stream) {
		defer stream.Close()
		peerID := stream.Conn().RemotePeer()

		for {
			ev := &event.Event{}

			err := event.ReadStream(stream, ev)
			if err != nil && xerrors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				log.Printf("%s error read message from stream: %+v", peerID, err)
				return
			}

			// recvs.Append(ev.Path)
			switch ev.Op {
			case event.Read:
				if err := ev.Read(); err != nil {
					log.Printf("%s error read event from stream: %+v", peerID, err)
					return
				}
				// recvDispChanged(ev.Path)
				if err := event.WriteStream(stream, ev); err != nil {
					log.Printf("%s error write event to stream: %+v", peerID, err)
					return
				}
			default:
				log.Printf("%s operator is not supported: %s ", peerID, ev.Op)
				return
			}
			// time.AfterFunc(time.Second, func() { recvs.Remove(ev.Path) })
		}
	}
}

func SnapWatcher(nd *p2p.Node, syncDir string) {
	var err error

	if syncDir == "" {
		if syncDir, err = os.Getwd(); err != nil {
			log.Fatalf("pwd: %+v\n", err)
		}
	}
	syncDir, err = filepath.Abs(syncDir)
	if err != nil {
		log.Fatalf("abs path: %+v\n", err)
	}

	for {
		kv := <-nd.DSPutCh
		snap, err := Restore(kv.B)
		if err != nil {
			log.Printf("restore(snap) failed: %+v\n", err)
			continue
		}
		if nd.Host.ID() == snap.PeerID {
			continue // myself
		}
		if !lo.Contains(p2p.Peers, snap.PeerID) {
			println("caught other")
			continue
		}

		diff, err := snap.Difference(syncDir)
		if err != nil {
			log.Printf("diff(snap) failed: %+v\n", err)
			continue
		}
		for _, meta := range diff.Adds {
			if meta.IsDir {
				continue
			}

			ev, err := notifyRead(nd.Host, snap.PeerID, meta.Path)
			if err != nil {
				log.Printf("notifyRead(Add) failed: %+v\n", err)
				continue
			}
			fmt.Printf("ADD(%s) %d: %s\n", ev.Op.String(), len(ev.Data), ev.Path)
		}
		for _, meta := range diff.Modifies {
			if meta.IsDir {
				continue
			}

			ev, err := notifyRead(nd.Host, snap.PeerID, meta.Path)
			if err != nil {
				log.Printf("notifyRead(Modify) failed: %+v\n", err)
				continue
			}
			fmt.Printf("MOD(%s) %d: %s\n", ev.Op.String(), len(ev.Data), ev.Path)
		}
		// for _, meta := range diff.Deletes {
		//  if meta.IsDir {
		//    continue
		//  }
		//  // TODO: delete to meta.Path
		// }
	}
}
