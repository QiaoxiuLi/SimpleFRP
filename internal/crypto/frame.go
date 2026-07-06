package crypto

import (
	"encoding/binary"
	"fmt"
	"io"
)

const MaxFrameSize = 4 * 1024 * 1024

func WriteEncryptedFrame(w io.Writer, key, plaintext []byte) error {
	sealed, err := Seal(key, plaintext)
	if err != nil {
		return err
	}
	if len(sealed) > MaxFrameSize {
		return fmt.Errorf("frame exceeds maximum size")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(sealed)))
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err = w.Write(sealed)
	return err
}

func ReadEncryptedFrame(r io.Reader, key []byte) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(header[:])
	if n == 0 || n > MaxFrameSize {
		return nil, fmt.Errorf("invalid encrypted frame size")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return Open(key, buf)
}
