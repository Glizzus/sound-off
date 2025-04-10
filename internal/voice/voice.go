package voice

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os/exec"

	"github.com/bwmarrin/discordgo"
	"github.com/jogramming/dca"
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

type VoiceChannelFunc func(*discordgo.Session, *discordgo.VoiceConnection) error

// WithVoiceChannel is a utility function
// that joins a voice channel and executes a callback.
// It handles the voice state updates for you.
func WithVoiceChannel(s *discordgo.Session, guildId, channelID string, callback VoiceChannelFunc) error {
	log.Printf("ID: " + s.State.User.ID)
	log.Printf("Channel ID: " + channelID)
	voiceConn, err := s.ChannelVoiceJoin(guildId, channelID, false, true)
	if err != nil {
		return fmt.Errorf("unable to join the voice channel: %w", err)
	}

	if err := voiceConn.Speaking(true); err != nil {
		return fmt.Errorf("error setting speaking state to 'true': %w", err)
	}
	defer func() {
		if err := voiceConn.Speaking(false); err != nil {
			slog.Error("failed to stop speaking", "error", err)
		}

		if err := voiceConn.Disconnect(); err != nil {
			slog.Error("failed to disconnect", "error", err)
		}
	}()

	if err = callback(s, voiceConn); err != nil {
		return fmt.Errorf("error executing callback: %w", err)
	}

	return nil
}

func StreamDCAOnTheFly(ctx context.Context, audioURL string) (*dca.EncodeSession, error) {
	// TODO: Require absolute paths for ffmpeg
	ffmpeg := exec.CommandContext(ctx, "ffmpeg",
		"-i", audioURL,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"pipe:1",
	)

	ffmpegStdout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("unable to pipe output of ffmpeg to stdout: %w", err)
	}

	ffmpeg.Stderr = nil

	if err := ffmpeg.Start(); err != nil {
		return nil, fmt.Errorf("unable to start ffmpeg process: %w", err)
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "audio"
	options.Volume = 256

	encodeSession, err := dca.EncodeMem(ffmpegStdout, options)
	if err != nil {
		return nil, fmt.Errorf("unable to encode dca from memory: %w", err)
	}

	return encodeSession, nil
}
