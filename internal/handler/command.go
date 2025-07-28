package handler

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var baseAddCommandOptions = []*discordgo.ApplicationCommandOption{
	{
		Name:        "cron",
		Type:        discordgo.ApplicationCommandOptionString,
		Description: "The cron expression for the soundcron.",
		Required:    false,
	},
	{
		Name:        "name",
		Type:        discordgo.ApplicationCommandOptionString,
		Description: "The name of the soundcron. Defaults to the file name if not provided.",
		Required:    false,
	},
}

var fileAddOptions = append([]*discordgo.ApplicationCommandOption{
	{
		Name:        "audio",
		Type:        discordgo.ApplicationCommandOptionAttachment,
		Description: "The file to play when the soundcron runs.",
		Required:    true,
	},
}, baseAddCommandOptions...)

// Commands is a list of all the commands the bot can handle.
// This is used to register the commands with Discord.
var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "soundcron",
		Description: "Manage and work with soundcrons",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "list",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "List all soundcrons",
			},
			{
				Name:        "add",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Description: "Add a soundcron to this server",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "file",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Add a soundcron using a file attachment.",
						Options:     fileAddOptions,
					},
				},
			},
		},
	},
}

func EstablishCommands(s *discordgo.Session, guildID string) error {
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, Commands)
	if err != nil {
		return fmt.Errorf("failed to establish commands: %w", err)
	}
	return nil
}
