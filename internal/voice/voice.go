package voice

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/dca"
)

// MaxAttendedChannel returns the channel with the most members in it.
// This returns nil if no channel has any members.
func MaxAttendedChannel(channels []*discordgo.Channel) *discordgo.Channel {
	var maxAttendedChannel *discordgo.Channel
	maxAttended := -1

	for _, channel := range channels {
		if channel.Type != discordgo.ChannelTypeGuildVoice {
			continue
		}

		if len(channel.Members) > maxAttended {
			maxAttendedChannel = channel
			maxAttended = len(channel.Members)
		}
	}

	return maxAttendedChannel
}

// VoiceChannelFunc is a function type that takes a discordgo session and a voice connection.
type VoiceChannelFunc func(*discordgo.Session, *discordgo.VoiceConnection) error

// WithVoiceChannel is a utility function
// that joins a voice channel and executes a callback.
// It handles the voice state updates for you.
func WithVoiceChannel(s *discordgo.Session, guildId, channelID string, callback VoiceChannelFunc) error {
	voiceConn, err := s.ChannelVoiceJoin(guildId, channelID, false, true)
	if err != nil {
		return fmt.Errorf("unable to join the voice channel: %w", err)
	}

	defer func() {
		if err := voiceConn.Speaking(false); err != nil {
			slog.Error("failed to stop speaking", "error", err)
		}

		if err := voiceConn.Disconnect(); err != nil {
			slog.Error("failed to disconnect", "error", err)
		}
	}()

	if err := voiceConn.Speaking(true); err != nil {
		return fmt.Errorf("error setting speaking state to 'true': %w", err)
	}

	if err = callback(s, voiceConn); err != nil {
		return fmt.Errorf("error executing callback: %w", err)
	}

	return nil
}

type DCAStreamer interface {
	StreamDCAOnTheFly(ctx context.Context, audioURL string) (*dca.EncodeSession, error)
}

type FFmpegDCAStreamer struct {
	urlReader URLReader
}

func NewFFmpegDCAStreamer(urlReader URLReader) *FFmpegDCAStreamer {
	return &FFmpegDCAStreamer{
		urlReader: urlReader,
	}
}

type URLReader interface {
	ReadURL(ctx context.Context, url string) (io.ReadCloser, error)
}

type HTTPURLReader struct {
	Client *http.Client
}

func (r *HTTPURLReader) ReadURL(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error response from server: %s", resp.Status)
	}

	return resp.Body, nil
}

func (s *FFmpegDCAStreamer) StreamDCAOnTheFly(ctx context.Context, audioURL string) (*dca.EncodeSession, error) {
	options := dca.StdEncodeOptions
	options.Bitrate = 96

	encodeSession, err := dca.EncodeFile(audioURL, options)
	if err != nil {
		return nil, fmt.Errorf("error encoding audio: %w", err)
	}

	return encodeSession, nil
}
