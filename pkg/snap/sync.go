package snap

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rjeczalik/notify"
	"github.com/samber/lo"
	"golang.org/x/xerrors"

	"github.com/threecorp/peerdrive/pkg/dev"
	"github.com/threecorp/peerdrive/pkg/event"
	"github.com/threecorp/peerdrive/pkg/p2p"
)

const Protocol = "/peerdrive/snap/1.0.0"

var (
	syncs = &dev.SafeSlice[string]{}
	recvs = &dev.SafeSlice[string]{}
)

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

			switch ev.Op {
			case event.Read:
				if err := ev.Read(); err != nil {
					log.Printf("%s error read event from stream: %+v", peerID, err)
					return
				}
				event.DispRecver(ev)
				if err := event.WriteStream(stream, ev); err != nil {
					log.Printf("%s error write event to stream: %+v", peerID, err)
					return
				}
			default:
				log.Printf("%s operator is not supported: %s ", peerID, ev.Op)
				return
			}
		}
	}
}

func SnapWatcher(nd *p2p.Node, syncDir string) {
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
			ev.Op = event.Write

			recvs.Append(ev.Path)
			if err := ev.Write(); err != nil {
				log.Printf("write read stream(Add) failed: %+v\n", err)
			}
			time.AfterFunc(time.Second, func() { recvs.Remove(ev.Path) })

			event.DispRecver(ev)
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
			ev.Op = event.Write

			recvs.Append(ev.Path)
			if err := ev.Write(); err != nil {
				log.Printf("write read stream(Modify) failed: %+v\n", err)
			}
			time.AfterFunc(time.Second, func() { recvs.Remove(ev.Path) })

			event.DispRecver(ev)
		}
		for _, meta := range diff.Deletes {
			if meta.IsDir {
				continue
			}
			ev := &event.Event{Op: event.Remove, Path: meta.Path}

			recvs.Append(ev.Path)
			if err := ev.Remove(); err != nil {
				log.Printf("delete file(Remove) failed: %+v\n", err)
			}
			time.AfterFunc(time.Second, func() { recvs.Remove(ev.Path) })

			event.DispRecver(ev)
		}
	}
}

func SyncWatcher(nd *p2p.Node, syncDir string) {
	nCh := make(chan notify.EventInfo)

	if err := notify.Watch(fmt.Sprintf("%s/...", syncDir), nCh, notify.All); err != nil {
		log.Fatalf("start watcher: %+v\n", err)
	}
	defer notify.Stop(nCh)

	for ev := range nCh {
		relPath := dev.RelativePath(syncDir, ev.Path()) // basename := filepath.Base(ev.Path())

		if syncs.Contains(relPath) {
			continue // fmt.Printf("syncs: %s\n", relPath)
		}
		if recvs.Contains(relPath) {
			continue // fmt.Printf("recvs: %s\n", relPath)
		}
		ignores := lo.Filter(dev.IgnoreNames, func(ig string, _ int) bool {
			return strings.HasPrefix(relPath, ig)
		})
		if len(ignores) != 0 {
			continue // fmt.Printf("ignores: %s\n", relPath)
		}

		syncs.Append(relPath)
		switch ev.Event() {
		case notify.Create:
			event.DispSendCreated(relPath)
		case notify.Remove:
			event.DispSendRemoved(relPath)
		case notify.Write:
			event.DispSendWritten(relPath)
		case notify.Rename:
			event.DispSendRenamed(relPath)
		}
		time.AfterFunc(time.Second, func() { syncs.Remove(relPath) })

		var (
			lastTime time.Time
			interval = 3 * time.Second
		)

		elapsed := time.Since(lastTime)
		if elapsed < interval {
			continue
		}
		dev.UntilWritten(ev.Path())
		if err := snapsnap(nd, syncDir); err != nil {
			log.Printf("send snapshot: %+v\n", err)
		}

		lastTime = time.Now()
	}
}

func snapsnap(nd *p2p.Node, syncDir string) error {
	sshot, err := Snapshot(nd.Host.ID(), syncDir)
	if err != nil {
		return xerrors.Errorf("snapshot: %w", err)
	}
	data, err := sshot.Marshal()
	if err != nil {
		return xerrors.Errorf("snapshot Marshal: %w", err)
	}
	if err := nd.DS.Put(context.Background(), SnapKey, data); err != nil {
		return xerrors.Errorf("snapshot ds.Put: %w", err)
	}

	return nil
}
