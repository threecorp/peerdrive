package event

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"golang.org/x/xerrors"
)

type Op uint

const (
	Write Op = iota
	Read
	Remove
)

var ops = map[Op]string{
	Write:  "WRITE",
	Read:   "READ",
	Remove: "REMOVE",
}

func (e Op) String() string {
	if op, ok := ops[e]; ok {
		return op
	}
	return "???"
}

type Event struct {
	Op
	Path string
	Data []byte
	Time time.Time
}

func (ev *Event) Write() error {
	// Create peer's dir
	dir := path.Dir(ev.Path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return xerrors.Errorf("%s error mkdirAll %s: %w", ev.String(), dir, err)
	}
	// Create peer's file
	f, err := os.Create(ev.Path)
	if err != nil {
		return xerrors.Errorf("%s error open %s: %w", ev.String(), ev.Path, err)
	}
	defer f.Close()

	// Write a data to peer's file
	if _, err := f.Write(ev.Data); err != nil {
		return xerrors.Errorf("%s error write data: %w", ev.String(), err)
	}
	// Write a data to peer's file modtime
	if !ev.Time.IsZero() {
		if err := os.Chtimes(ev.Path, time.Now(), ev.Time); err != nil {
			return xerrors.Errorf("%s error write modtime: %w", ev.String(), err)
		}
	}

	return nil
}

func (ev *Event) Read() error {
	if len(ev.Data) != 0 {
		return xerrors.Errorf("%s error Data is not empty", ev.String())
	}

	// Open local's file
	// Read a data to local's file
	data, err := ioutil.ReadFile(ev.Path)
	if err != nil {
		return xerrors.Errorf("%s error read %s: %w", ev.String(), ev.Path, err)
	}
	ev.Data = data

	fi, err := os.Stat(ev.Path)
	if err != nil {
		return xerrors.Errorf("%s error info %s: %w", ev.String(), ev.Path, err)
	}
	ev.Time = fi.ModTime()

	return nil
}

func (ev *Event) Remove() error {
	if err := os.Remove(ev.Path); err != nil {
		return xerrors.Errorf("%s error remove %s: %w", ev.String(), ev.Path, err)
	}
	return nil
}
