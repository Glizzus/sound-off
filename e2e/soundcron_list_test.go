package e2e_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/e2e"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/handler"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/google/go-cmp/cmp"
)

func seedTestData(t *testing.T, repo *repository.PostgresSoundCronRepository) {
	const guildID = "74241007174813750"

	soundCrons := []repository.SoundCron{
		{
			ID:      "302808d9-141e-410d-a69d-2418ad15b5de",
			Name:    "Everything She Wants (Wham!)",
			GuildID: guildID,
			Cron:    "*/5 * * * *",
		},
		{
			ID:      "8597e24a-f204-4c88-bad0-fe0ab9a73ff1",
			Name:    "Take On Me (A-ha)",
			GuildID: guildID,
			Cron:    "*/10 * * * *",
		},
	}
	for _, soundCron := range soundCrons {
		err := repo.Save(t.Context(), soundCron)
		if err != nil {
			t.Fatalf("failed to save SoundCron: %v", err)
		}
	}
}

type determinsticIDGenerator struct{}

func (d *determinsticIDGenerator) Next() (string, error) {
	return "determinism", nil
}

var _ generator.Generator[string] = (*determinsticIDGenerator)(nil)

func TestSoundCronList_NoSoundCrons(t *testing.T) {
	connStr := e2e.UsePostgres(t)
	repo := e2e.GetRepository(t, connStr)

	slashCommandInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "soundcron",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name:  "list",
						Type:  discordgo.ApplicationCommandOptionSubCommand,
						Value: "list",
					},
				},
			},
			GuildID: "00000000000000000",
		},
	}

	session := &mockSession{}

	handler := handler.NewInteractionHandler(repo, nil, &determinsticIDGenerator{})
	handler(session, slashCommandInteraction)

	expected := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "No soundcrons found",
		},
	}

	diff := cmp.Diff(expected, session.Resp)
	if diff != "" {
		t.Errorf("session mismatch (-want +got):\n%s", diff)
	}
}

func TestSoundCronList_HappyPath(t *testing.T) {
	connStr := e2e.UsePostgres(t)
	repo := e2e.GetRepository(t, connStr)
	seedTestData(t, repo)

	t.Run("lists soundcrons with select menu", func(t *testing.T) {
		slashCommandInteraction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "soundcron",
					Options: []*discordgo.ApplicationCommandInteractionDataOption{
						{
							Name:  "list",
							Type:  discordgo.ApplicationCommandOptionSubCommand,
							Value: "list",
						},
					},
				},
				GuildID: "74241007174813750",
			},
		}

		session := &mockSession{}

		handler := handler.NewInteractionHandler(repo, nil, &determinsticIDGenerator{})
		handler(session, slashCommandInteraction)

		expected := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Choose a soundcron:",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								CustomID:    "soundcron_select_menu:determinism",
								Placeholder: "Select a soundcron",
								MinValues:   &[]int{1}[0],
								MaxValues:   1,
								Options: []discordgo.SelectMenuOption{
									{
										Label: "Everything She Wants (Wham!)",
										Value: "302808d9-141e-410d-a69d-2418ad15b5de",
									},
									{
										Label: "Take On Me (A-ha)",
										Value: "8597e24a-f204-4c88-bad0-fe0ab9a73ff1",
									},
								},
							},
						},
					},
				},
			},
		}

		diff := cmp.Diff(expected, session.Resp)
		if diff != "" {
			t.Errorf("session mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("shows actions when soundcron is selected", func(t *testing.T) {})
}
