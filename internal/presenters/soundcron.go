package presenters

import (
	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/repository"
)

var noSoundCronFoundResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "No soundcrons found",
	},
}

func soundCronToSelectMenuOption(sc repository.SoundCron) discordgo.SelectMenuOption {
	return discordgo.SelectMenuOption{
		Label: sc.Name,
		Value: sc.ID,
	}
}

var soundCronSelectMinValues = 1

const ComponentIDSoundCronSelect = "soundcron_select_menu"

func buildSoundCronSelectMenu(soundCrons []repository.SoundCron, instanceID string) *discordgo.InteractionResponse {
	var options []discordgo.SelectMenuOption
	for _, sc := range soundCrons {
		options = append(options, soundCronToSelectMenuOption(sc))
	}

	menu := discordgo.SelectMenu{
		CustomID:    ComponentIDSoundCronSelect + ":" + instanceID,
		Placeholder: "Select a soundcron",
		MinValues:   &soundCronSelectMinValues,
		MaxValues:   1,
		Options:     options,
	}

	row := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			menu,
		},
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "**Current Soundcrons** _(select for more details)_",
			Components: []discordgo.MessageComponent{
				row,
			},
		},
	}
}

func BuildListSoundCronsResponse(soundCrons []repository.SoundCron, instanceID string) *discordgo.InteractionResponse {
	if len(soundCrons) == 0 {
		return noSoundCronFoundResponse
	}

	return buildSoundCronSelectMenu(soundCrons, instanceID)
}

const (
	ComponentIDSoundCronEdit     = "soundcron_edit"
	ComponentIDSoundCronDelete   = "soundcron_delete"
)

// SoundCronListActionsMenu builds the response for the soundcron list actions menu.
// This is what is sent after the user selects a soundcron from the select menu.
func SoundCronListActionsMenu(instanceID, name string) *discordgo.InteractionResponse {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: name,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Edit",
							Style:    discordgo.SecondaryButton,
							CustomID: ComponentIDSoundCronEdit + ":" + instanceID,
						},
						discordgo.Button{
							Label:    "Delete",
							Style:    discordgo.DangerButton,
							CustomID: ComponentIDSoundCronDelete + ":" + instanceID,
						},
					},
				},
			},
		},
	}
	return response
}
