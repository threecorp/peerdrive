package snap

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/xerrors"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/threecorp/peerdrive/pkg/dev"
)

const (
	SnapName = "snap"
)

var (
	SnapKey = datastore.NewKey(SnapName)
)

type (
	Meta struct {
		Path    string
		Name    string
		Size    int64
		ModTime time.Time
		IsDir   bool
	}
	Diff struct {
		Adds     []*Meta
		Deletes  []*Meta
		Modifies []*Meta
	}
	Snap struct {
		PeerID peer.ID
		Metas  []*Meta
	}
)

func Snapshot(peerID peer.ID, dir string) (*Snap, error) {
	metas, err := makeMetas(dir)
	if err != nil {
		return nil, xerrors.Errorf("Snapshot(%s): %w", dir, err)
	}
	return &Snap{PeerID: peerID, Metas: metas}, nil
}

func Restore(data []byte) (*Snap, error) {
	snap := &Snap{}
	if err := snap.Unmarshal(data); err != nil {
		return nil, xerrors.Errorf("Restore(%d bytes): %w", len(data), err)
	}
	return snap, nil
}

func (s *Snap) Difference(dir string) (*Diff, error) {
	locals, err := makeMetas(dir)
	if err != nil {
		return nil, xerrors.Errorf("Difference(%s): %w", dir, err)
	}
	return calcDiff(locals, s.Metas), nil
}

// Marshal encodes the Meta object into a byte slice using gob
func (s *Snap) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal decodes a byte slice into a Meta object using gob
func (s *Snap) Unmarshal(data []byte) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(&s)
}

func makeMetas(dir string) ([]*Meta, error) {
	metas := []*Meta{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		for _, ig := range dev.IgnoreNames {
			if info.IsDir() && info.Name() == ig {
				return filepath.SkipDir
			}
		}
		if !info.IsDir() && info.Name() == dev.PrivateKeyName {
			return nil
		}

		metas = append(metas, &Meta{
			Path:    dev.RelativePath(dir, path),
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return metas, nil
}

func calcDiff(local, remote []*Meta) *Diff {
	lmap := make(map[string]*Meta)
	rmap := make(map[string]*Meta)

	for _, snap := range local {
		lmap[snap.Path] = snap
	}
	for _, snap := range remote {
		rmap[snap.Path] = snap
	}

	diff := &Diff{}
	for path, lsnap := range lmap {
		rsnap, ok := rmap[path]

		if !ok {
			diff.Deletes = append(diff.Deletes, lsnap)
		} else if lsnap.Size != rsnap.Size {
			diff.Modifies = append(diff.Modifies, lsnap)
		} else if !lsnap.ModTime.Equal(rsnap.ModTime) {
			diff.Modifies = append(diff.Modifies, lsnap)
		}
	}

	for path, rsnap := range rmap {
		if _, ok := lmap[path]; !ok {
			diff.Adds = append(diff.Adds, rsnap)
		}
	}

	return diff
}
