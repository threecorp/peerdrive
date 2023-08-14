package event

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"

	"golang.org/x/xerrors"
)

func WriteStream(stream io.Writer, ev *Event) error {
	b := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(b).Encode(ev); err != nil {
		return xerrors.Errorf("%s error sending message encode: %w", err)
	}
	buf := b.Bytes()

	packetSize := make([]byte, 4)
	binary.BigEndian.PutUint32(packetSize, uint32(len(buf)))

	writer := bufio.NewWriter(stream)
	if _, err := writer.Write(packetSize); err != nil {
		return xerrors.Errorf("%s error sending message length: %w", err)
	}
	if _, err := writer.Write(buf); err != nil {
		return xerrors.Errorf("%s error sending message: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return xerrors.Errorf("%s error flushing writer: %w", err)
	}

	return nil
}

func ReadStream(stream io.Reader, ev *Event) error {
	packetSize := make([]byte, 4)

	if _, err := io.ReadFull(stream, packetSize); err != nil {
		return xerrors.Errorf("error read length from stream: %+v", err)
	}

	data := make([]byte, binary.BigEndian.Uint32(packetSize))
	if _, err := io.ReadFull(stream, data); err != nil {
		return xerrors.Errorf("error read message from stream: %+v", err)
	}
	if err := gob.NewDecoder(bytes.NewBuffer(data)).Decode(ev); err != nil {
		return xerrors.Errorf("error read message from stream: %+v", err)
	}

	return nil
}
