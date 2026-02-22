package opus

import (
	"bufio"
	"encoding/binary"
	"io"
)

// FrameReader reads length-prefixed Opus frames from an io.Reader.
type FrameReader struct {
	r io.Reader
	hdr [2]byte
}

// NewFrameReader returns a new FrameReader that reads from r.
// The reader is buffered to minimize small reads against the underlying source.
func NewFrameReader(r io.Reader) *FrameReader {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &FrameReader{r: br}
}

// ReadFrame reads and returns the next raw Opus frame.
// Returns io.EOF when there are no more frames.
func (f *FrameReader) ReadFrame() ([]byte, error) {
	if _, err := io.ReadFull(f.r, f.hdr[:]); err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint16(f.hdr[:])

	frame := make([]byte, size)
	if _, err := io.ReadFull(f.r, frame); err != nil {
		return nil, err
	}
	return frame, nil
}
