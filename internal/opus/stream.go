package opus

import (
	"errors"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
)

var ErrVoiceConnClosed = errors.New("voice connection send timeout")

// StreamToVoice reads Opus frames from source and sends them to the Discord
// voice connection. It blocks until all frames are sent or an error occurs.
// Returns nil on clean EOF.
func StreamToVoice(source *FrameReader, vc *discordgo.VoiceConnection) error {
	for {
		frame, err := source.ReadFrame()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}

		timer := time.NewTimer(time.Minute)
		select {
		case vc.OpusSend <- frame:
			timer.Stop()
		case <-timer.C:
			return ErrVoiceConnClosed
		}
	}
}
