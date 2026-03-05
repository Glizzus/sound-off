package opus

import (
	"errors"
	"fmt"
	"io"

	"github.com/disgoorg/disgo/voice"
)

// StreamToVoice reads Opus frames from source and sends them to the Discord
// voice connection. It blocks until all frames are sent or an error occurs.
// Returns nil on clean EOF.
func StreamToVoice(source *FrameReader, vc voice.Conn) error {
	for {
		frame, err := source.ReadFrame()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}

		_, err = vc.UDP().Write(frame)
		if err != nil {
			return fmt.Errorf("failed to write opus frame: %w", err)
		}
	}
}