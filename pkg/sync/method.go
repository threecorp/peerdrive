package sync

import (
	"context"
	"io/ioutil"

	"golang.org/x/xerrors"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/samber/lo"

	"github.com/threecorp/peerdrive/pkg/event"
	"github.com/threecorp/peerdrive/pkg/p2p"
)

func notifyWrite(h host.Host, path, relPath string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return xerrors.Errorf("notify copy failed: %w", err)
	}

	ev := &event.Event{Op: event.Write, Path: relPath, Data: data}
	return writeStreams(context.Background(), h, Protocol, ev)
}

func notifyDelete(h host.Host, relPath string) error {
	ev := &event.Event{Op: event.Remove, Path: relPath}
	return writeStreams(context.Background(), h, Protocol, ev)
}

func writeStreams(ctx context.Context, h host.Host, protocol protocol.ID, ev *event.Event) error {
	for _, peerID := range lo.Uniq(p2p.Peers) {
		if err := writeStream(ctx, h, protocol, peerID, ev); err != nil {
			return xerrors.Errorf("%s write stream failed: %w", peerID, err)
		}
	}

	return nil
}

func writeStream(ctx context.Context, h host.Host, protocol protocol.ID, peerID peer.ID, ev *event.Event) error {
	stream, err := h.NewStream(ctx, peerID, protocol)
	if err != nil {
		return xerrors.Errorf("%s stream open failed: %w", peerID, err)
	}
	defer stream.Close()

	if err := event.WriteStream(stream, ev); err != nil {
		return xerrors.Errorf("%s error sending message: %w", peerID, err)
	}

	return nil
}
