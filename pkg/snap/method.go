package snap

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/threecorp/peerdrive/pkg/event"
)

func notifyRead(h host.Host, peerID peer.ID, relPath string) (*event.Event, error) {
	ev := &event.Event{Op: event.Read, Path: relPath}
	return rwStream(context.Background(), h, Protocol, peerID, ev)
}

func rwStream(ctx context.Context, h host.Host, protocol protocol.ID, peerID peer.ID, ev *event.Event) (*event.Event, error) {
	println(1)
	stream, err := h.NewStream(ctx, peerID, protocol)
	if err != nil {
		println(2)
		return nil, xerrors.Errorf("%s stream open failed: %w", peerID, err)
	}
	defer stream.Close()
	println(3)

	if err := event.WriteStream(stream, ev); err != nil {
		println(4)
		return nil, xerrors.Errorf("%s error sending message: %w", peerID, err)
	}
	println(5)
	if err := event.ReadStream(stream, ev); err != nil {
		println(6)
		return nil, xerrors.Errorf("%s error reading message: %w", peerID, err)
	}

	println(7)
	return ev, nil
}
