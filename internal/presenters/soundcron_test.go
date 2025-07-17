package presenters_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/presenters"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/google/go-cmp/cmp"
)

func TestBuildListSoundCronsResponse(t *testing.T) {
	tests := []struct {
		name  string
		input []repository.SoundCron
		want  *discordgo.InteractionResponse
	}{
		{
			name:  "no soundcrons",
			input: []repository.SoundCron{},
			want: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "No soundcrons found",
				},
			},
		},
		{
			name: "any soundcrons",
			input: []repository.SoundCron{
				{
					ID:   "test-sc-1",
					Name: "Test SoundCron 1",
				},
				{
					ID:   "test-sc-2",
					Name: "Test SoundCron 2",
				},
			},
			want: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "**Current Soundcrons** _(select for more details)_",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "soundcron_select_menu:test-sc-1",
									Placeholder: "Select a soundcron",
									MinValues:   &[]int{1}[0],
									MaxValues:   1,
									Options: []discordgo.SelectMenuOption{
										{
											Label: "Test SoundCron 1",
											Value: "test-sc-1",
										},
										{
											Label: "Test SoundCron 2",
											Value: "test-sc-2",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := presenters.BuildListSoundCronsResponse(tt.input, "test-sc-1")
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Errorf("BuildListSoundCronsResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSoundCronListActionsMenu(t *testing.T) {
	tests := []struct {
		name  string
		input repository.SoundCron
		want  *discordgo.InteractionResponse
	}{
		{
			name: "any soundcron",
			input: repository.SoundCron{
				ID:   "test-sc-1",
				Name: "Test SoundCron 1",
			},
			want: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Test SoundCron 1",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Edit",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_edit:test-sc-1",
								},
								discordgo.Button{
									Label:    "Delete",
									Style:    discordgo.DangerButton,
									CustomID: "soundcron_delete:test-sc-1",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := presenters.SoundCronListActionsMenu(tt.input.ID, tt.input.Name)
			diff := cmp.Diff(got, tt.want)
			if diff != "" {
				t.Errorf("SoundCronListActionsMenu() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
