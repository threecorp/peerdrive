package sync

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"golang.org/x/xerrors"

	"github.com/radovskyb/watcher"

	"github.com/threecorp/peerdrive/pkg/dev"
	"github.com/threecorp/peerdrive/pkg/event"
	"github.com/threecorp/peerdrive/pkg/p2p"
	"github.com/threecorp/peerdrive/pkg/snap"
)

const Protocol = "/peerdrive/sync/1.0.0"

var (
	syncs = &SafeSlice[string]{}
	recvs = &SafeSlice[string]{}
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
			if ev == nil {
				log.Printf("%s error read message from stream: %+v", peerID, err)
				return
			}

			recvs.Append(ev.Path)
			switch ev.Op {
			case event.Write:
				err = ev.Write()
				recvDispChanged(ev.Path)
			case event.Remove:
				err = ev.Remove()
				recvDispDeleted(ev.Path)
			default:
				log.Printf("%s operator is not supported: %s ", peerID, ev.Op)
				return
			}
			time.AfterFunc(time.Second, func() { recvs.Remove(ev.Path) })
			if err != nil {
				log.Printf("%s error operate message from stream: %+v", peerID, err)
				return
			}
		}
	}
}

func SyncWatcher(nd *p2p.Node, syncDir string) {
	var (
		h       = nd.Host
		w       = watcher.New()
		watchCh = make(chan watcher.Event, 100)
		err     error
	)

	if syncDir == "" {
		if syncDir, err = os.Getwd(); err != nil {
			log.Fatalf("pwd: %+v\n", err)
		}
	}
	syncDir, err = filepath.Abs(syncDir)
	if err != nil {
		log.Fatalf("abs path: %+v\n", err)
	}

	go func() {
		for {
			select {
			case ev := <-w.Event:
				if strings.HasPrefix(ev.Name(), ".") {
					break
				}
				if ev.Op == watcher.Chmod {
					break
				}
				if ev.IsDir() {
					break
				}
				relPath, _ := paths(syncDir, ev)
				if syncs.Contains(relPath) {
					break
				}
				if recvs.Contains(relPath) {
					break
				}

				watchCh <- ev
			case err := <-w.Error:
				log.Fatalf("watcher: %+v\n", err)
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		for {
			ev := <-watchCh
			untilWritten(ev.Path)

			relPath, oldPath := paths(syncDir, ev)
			syncs.Append(relPath)

			if runtime.GOOS == "darwin" {
				sshot, err := snap.Snapshot(h.ID(), syncDir)
				logFatal(err)

				data, err := sshot.Marshal()
				logFatal(err)

				logFatal(nd.DS.Put(context.Background(), snap.SnapKey, data))
			}

			switch ev.Op {
			case watcher.Move, watcher.Rename:
				logFatal(notifyWrite(h, ev.Path, relPath))
				sendDispChanged(relPath)
				logFatal(notifyDelete(h, oldPath))
				sendDispDeleted(relPath)
			case watcher.Create, watcher.Write:
				logFatal(notifyWrite(h, ev.Path, relPath))
				sendDispChanged(relPath)
			case watcher.Remove:
				logFatal(notifyDelete(h, relPath))
				sendDispDeleted(relPath)
			}

			time.AfterFunc(time.Second, func() { syncs.Remove(relPath) })
		}
	}()

	if err := w.AddRecursive("./"); err != nil {
		log.Fatalf("recursive watcher: %+v\n", err)
	}
	if err := w.Ignore(dev.IgnoreNames...); err != nil {
		log.Fatalf("ignore watcher: %+v\n", err)
	}
	if err := w.Start(time.Millisecond * 300); err != nil {
		log.Fatalf("start watcher: %+v\n", err)
	}
}
