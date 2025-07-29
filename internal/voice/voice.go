package voice

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// MaxAttendedVoiceChannel returns the ID of the voice channel with the most members.
// If there are no voice channels with members, it returns an empty string.
func MaxAttendedVoiceChannel(vcs []*discordgo.VoiceState) string {
	var counts = make(map[string]int)
	for _, vs := range vcs {
		if vs.ChannelID == "" {
			continue
		}
		counts[vs.ChannelID]++
	}
	var maxCount int
	var maxChannelID string
	for channelID, count := range counts {
		if count > maxCount {
			maxCount = count
			maxChannelID = channelID
		}
	}
	if maxCount == 0 {
		return ""
	}
	return maxChannelID
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
