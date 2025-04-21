package e2e_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/google/go-cmp/cmp"

	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/handler"
)

type mockSession struct {
	Called bool
	Resp   *discordgo.InteractionResponse
}

func (m *mockSession) InteractionRespond(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
	m.Called = true
	m.Resp = resp
	return nil
}

func (m *mockSession) InteractionResponseEdit(i *discordgo.Interaction, wh *discordgo.WebhookEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, nil
}

var _ handler.DiscordSession = (*mockSession)(nil)

func TestInteractionCreatePing(t *testing.T) {
	session := &mockSession{}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "ping",
			},
		},
	}

	handler := handler.NewInteractionHandler(nil, nil, &generator.UUIDV4Generator{})
	handler(session, interaction)

	expectedSession := &mockSession{
		Called: true,
		Resp: &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong!",
			},
		},
	}

	diff := cmp.Diff(expectedSession, session)
	if diff != "" {
		t.Errorf("session mismatch (-want +got):\n%s", diff)
	}
}
