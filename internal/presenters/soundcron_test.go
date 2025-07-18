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
					Content: "Current SoundCrons",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 1",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-1",
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 2",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-2",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "more than four soundcrons",
			input: []repository.SoundCron{
				{
					ID:   "test-sc-1",
					Name: "Test SoundCron 1",
					Cron: "0 0 * * *",
				},
				{
					ID:   "test-sc-2",
					Name: "Test SoundCron 2",
					Cron: "0 1 * * *",
				},
				{
					ID:   "test-sc-3",
					Name: "Test SoundCron 3",
					Cron: "0 2 * * *",
				},
				{
					ID:   "test-sc-4",
					Name: "Test SoundCron 4",
					Cron: "0 3 * * *",
				},
				{
					ID:   "test-sc-5",
					Name: "Test SoundCron 5",
					Cron: "0 4 * * *",
				},
			},
			want: &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Current SoundCrons",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 1",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-1",
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 2",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-2",
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 3",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-3",
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "Test SoundCron 4",
									Style:    discordgo.SecondaryButton,
									CustomID: "soundcron_select_menu:random-instance-id:test-sc-4",
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "soundcron_select_menu:random-instance-id",
									Placeholder: "More soundcrons...",
									MinValues:   &[]int{1}[0],
									MaxValues:   1,
									Options: []discordgo.SelectMenuOption{
										{
											Label: "Test SoundCron 5",
											Value: "test-sc-5",
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
			got := presenters.BuildListSoundCronsResponse(tt.input, "random-instance-id")
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
