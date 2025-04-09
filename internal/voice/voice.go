package voice

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// MaxAttendedChannel returns the channel with the most members in it.
// This returns nil if no channel has any members.
func MaxAttendedChannel(channels []*discordgo.Channel) *discordgo.Channel {
	var maxAttendedChannel *discordgo.Channel
	maxAttended := 0

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

type VoiceChannelFunc func(*discordgo.Session, *discordgo.VoiceConnection)

// WithVoiceChannel is a utility function
// that joins a voice channel and executes a callback.
// It handles the voice state updates for you.
func WithVoiceChannel(s *discordgo.Session, channelID string, callback VoiceChannelFunc) error {
	voiceConn, err := s.ChannelVoiceJoin(s.State.User.ID, channelID, false, true)
	if err != nil {
		return err
	}

	if err := voiceConn.Speaking(true); err != nil {
		return err
	}
	defer func() {
		if err := voiceConn.Speaking(false); err != nil {
			slog.Error("failed to stop speaking", "error", err)
		}
	}()

	callback(s, voiceConn)
	return nil
}
