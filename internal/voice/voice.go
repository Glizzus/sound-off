package voice

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
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
type VoiceChannelFunc func(voice.Conn) error

// WithVoiceChannel is a utility function
// that joins a voice channel and executes a callback.
// It handles the voice state updates for you.
func WithVoiceChannel(ctx context.Context,manager voice.Manager, guildIDStr, channelIDStr string, callback VoiceChannelFunc) error {
	guildId, err := snowflake.Parse(guildIDStr)
	if err != nil {
		return fmt.Errorf("invalid guild ID: %w", err)
	}

	channelID, err := snowflake.Parse(channelIDStr)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	conn := manager.CreateConn(guildId)
	const selfMute = false
	const selfDeaf = true
	err = conn.Open(ctx, channelID, selfMute, selfDeaf)
	defer conn.Close(ctx)

	if err := conn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone); err != nil {
		return fmt.Errorf("error setting speaking state to 'true': %w", err)
	}

	if err = callback(conn); err != nil {
		return fmt.Errorf("error executing callback: %w", err)
	}

	return nil
}
