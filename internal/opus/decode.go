package opus

import (
	"encoding/binary"
	"io"
)

// FrameReader reads length-prefixed Opus frames from an io.Reader.
type FrameReader struct {
	r io.Reader
}

// NewFrameReader returns a new FrameReader that reads from r.
func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{r: r}
}

// ReadFrame reads and returns the next raw Opus frame.
// Returns io.EOF when there are no more frames.
func (f *FrameReader) ReadFrame() ([]byte, error) {
	var size uint16
	if err := binary.Read(f.r, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	frame := make([]byte, size)
	if _, err := io.ReadFull(f.r, frame); err != nil {
		return nil, err
	}
	return frame, nil
}
