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

var soundCronListSlashCommandInteraction = &discordgo.InteractionCreate{
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

var expectedSoundCronListResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "Current SoundCrons",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Take On Me (A-ha)",
						Style:    discordgo.SecondaryButton,
						CustomID: "soundcron_select_menu:determinism:8597e24a-f204-4c88-bad0-fe0ab9a73ff1",
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Everything She Wants (Wham!)",
						Style:    discordgo.SecondaryButton,
						CustomID: "soundcron_select_menu:determinism:302808d9-141e-410d-a69d-2418ad15b5de",
					},
				},
			},
		},
	},
}

var soundCronListButtonInteraction = &discordgo.InteractionCreate{
	Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionMessageComponent,
		Data: discordgo.MessageComponentInteractionData{
			CustomID: "soundcron_select_menu:determinism:8597e24a-f204-4c88-bad0-fe0ab9a73ff1",
			ComponentType: discordgo.ButtonComponent,
		},
		GuildID: "74241007174813750",
	},
}

var expectedSoundCronListButtonResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "Take On Me (A-ha)",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "Edit",
						Style: discordgo.SecondaryButton,
						CustomID: "soundcron_edit:determinism",
					},
					discordgo.Button{
						Label: "Delete",
						Style: discordgo.DangerButton,
						CustomID: "soundcron_delete:determinism",
					},
				},
			},
		},
	},
}
					  

func TestSoundCronList(t *testing.T) {
	connStr := e2e.UsePostgres(t)
	repo := e2e.GetRepository(t, connStr)
	seedTestData(t, repo)

	handler := handler.NewInteractionHandler(repo, nil, &determinsticIDGenerator{})
	session := &mockSession{}

	handler(session, soundCronListSlashCommandInteraction)

	diff := cmp.Diff(expectedSoundCronListResponse, session.Resp)
	if diff != "" {
		t.Errorf("session mismatch (-want +got):\n%s", diff)
	}

	handler(session, soundCronListButtonInteraction)
	diff = cmp.Diff(expectedSoundCronListButtonResponse, session.Resp)
	if diff != "" {
		t.Errorf("session mismatch (-want +got):\n%s", diff)
	}
}
