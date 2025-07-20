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

func splitSoundCronsForList(soundCrons []repository.SoundCron) (buttons []repository.SoundCron, menu []repository.SoundCron) {
	if len(soundCrons) <= 4 {
		return soundCrons, nil
	}

	buttons = soundCrons[:4]
	menu = soundCrons[4:]

	return buttons, menu
}

func soundCronToButton(sc repository.SoundCron, instanceID string) discordgo.Button {
	label := sc.Name
	return discordgo.Button{
		Label:    label,
		Style:    discordgo.SecondaryButton,
		CustomID: ComponentIDSoundCronSelect + ":" + instanceID + ":" + sc.ID,
	}
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
	firstFour, rest := splitSoundCronsForList(soundCrons)

	rows := make([]discordgo.MessageComponent, 0, len(firstFour)+1)

	for _, sc := range firstFour {
		button := soundCronToButton(sc, instanceID)
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{button},
		})
	}

	if len(rest) > 0 {
		selectOptions := make([]discordgo.SelectMenuOption, 0, len(rest))
		for _, sc := range rest {
			selectOptions = append(selectOptions, soundCronToSelectMenuOption(sc))
		}

		menu := discordgo.SelectMenu{
			CustomID:    ComponentIDSoundCronSelect + ":" + instanceID,
			Placeholder: "More soundcrons...",
			MinValues:   &soundCronSelectMinValues,
			MaxValues:   1,
			Options:     selectOptions,
		}

		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Current SoundCrons",
			Components: rows,
		},
	}
}

// BuildListSoundCronsResponse builds the response for listing soundcrons.
// This response is structured to where the first 4 rows are a button representing the
// first 4 soundcrons, and the rest are in a select menu.
func BuildListSoundCronsResponse(soundCrons []repository.SoundCron, instanceID string) *discordgo.InteractionResponse {
	if len(soundCrons) == 0 {
		return noSoundCronFoundResponse
	}

	return buildSoundCronSelectMenu(soundCrons, instanceID)
}

const (
	ComponentIDSoundCronEdit   = "soundcron_edit"
	ComponentIDSoundCronDelete = "soundcron_delete"
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
