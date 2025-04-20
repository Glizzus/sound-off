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

func buildSoundCronSelectMenu(soundCrons []repository.SoundCron) *discordgo.InteractionResponse {
	var options []discordgo.SelectMenuOption
	for _, sc := range soundCrons {
		options = append(options, soundCronToSelectMenuOption(sc))
	}

	menu := discordgo.SelectMenu{
		CustomID:    ComponentIDSoundCronSelect,
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
			Content: "Choose a soundcron:",
			Components: []discordgo.MessageComponent{
				row,
			},
		},
	}
}

func BuildListSoundCronsResponse(soundCrons []repository.SoundCron) *discordgo.InteractionResponse {
	if len(soundCrons) == 0 {
		return noSoundCronFoundResponse
	}

	return buildSoundCronSelectMenu(soundCrons)
}
