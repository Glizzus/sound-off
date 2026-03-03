package handler_test

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/handler"
)

// mockSession captures the InteractionRespond call for inspection.
type mockSession struct {
	responded bool
	response  *discordgo.InteractionResponse
}

func (m *mockSession) InteractionRespond(
	_ *discordgo.Interaction,
	r *discordgo.InteractionResponse,
	_ ...discordgo.RequestOption,
) error {
	m.responded = true
	m.response = r
	return nil
}

func (m *mockSession) InteractionResponseEdit(
	_ *discordgo.Interaction,
	_ *discordgo.WebhookEdit,
	_ ...discordgo.RequestOption,
) (*discordgo.Message, error) {
	return nil, nil
}

// makeAutocompleteInteraction builds an InteractionCreate that looks like the
// user typing `prefix` into the timezone field of /soundcron add file.
func makeAutocompleteInteraction(prefix string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommandAutocomplete,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "soundcron",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name: "add",
						Type: discordgo.ApplicationCommandOptionSubCommandGroup,
						Options: []*discordgo.ApplicationCommandInteractionDataOption{
							{
								Name: "file",
								Type: discordgo.ApplicationCommandOptionSubCommand,
								Options: []*discordgo.ApplicationCommandInteractionDataOption{
									{
										Name:    "timezone",
										Type:    discordgo.ApplicationCommandOptionString,
										Value:   prefix,
										Focused: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func choiceNames(resp *discordgo.InteractionResponse) []string {
	names := make([]string, 0, len(resp.Data.Choices))
	for _, c := range resp.Data.Choices {
		names = append(names, c.Name)
	}
	return names
}

func TestHandleTimezoneAutocomplete_EmptyPrefix(t *testing.T) {
	s := &mockSession{}
	handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction(""))

	if !s.responded {
		t.Fatal("expected InteractionRespond to be called")
	}
	if s.response.Type != discordgo.InteractionApplicationCommandAutocompleteResult {
		t.Fatalf("expected autocomplete result type, got %v", s.response.Type)
	}

	// Empty prefix should return exactly the common timezones.
	names := choiceNames(s.response)
	if len(names) == 0 {
		t.Fatal("expected non-empty choices for empty prefix")
	}
	found := false
	for _, n := range names {
		if n == "UTC" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected UTC in common timezone choices, got: %v", names)
	}
}

func TestHandleTimezoneAutocomplete_PrefixSearch(t *testing.T) {
	s := &mockSession{}
	handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction("America"))

	if !s.responded {
		t.Fatal("expected InteractionRespond to be called")
	}

	names := choiceNames(s.response)
	if len(names) == 0 {
		t.Fatal("expected results for prefix 'America'")
	}
	for _, n := range names {
		if !strings.HasPrefix(n, "America") {
			t.Errorf("result %q does not have prefix 'America'", n)
		}
	}
}

func TestHandleTimezoneAutocomplete_ExactMatch(t *testing.T) {
	s := &mockSession{}
	handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction("America/New_York"))

	names := choiceNames(s.response)
	if len(names) == 0 {
		t.Fatal("expected at least one result for 'America/New_York'")
	}
	if names[0] != "America/New_York" {
		t.Errorf("expected first result to be 'America/New_York', got %q", names[0])
	}
}

func TestHandleTimezoneAutocomplete_NoMatch(t *testing.T) {
	s := &mockSession{}
	handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction("Nonexistent/Zone"))

	if !s.responded {
		t.Fatal("expected InteractionRespond to be called even with no matches")
	}
	names := choiceNames(s.response)
	if len(names) != 0 {
		t.Errorf("expected empty choices for non-matching prefix, got: %v", names)
	}
}

func TestHandleTimezoneAutocomplete_ResultsCappedAt25(t *testing.T) {
	s := &mockSession{}
	// "A" matches Africa/*, America/*, Antarctica/*, Arctic/*, Asia/*, Atlantic/*, Australia/* — well over 25
	handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction("A"))

	names := choiceNames(s.response)
	if len(names) > 25 {
		t.Errorf("expected at most 25 choices, got %d", len(names))
	}
}

func TestHandleTimezoneAutocomplete_WrongFocusedOption(t *testing.T) {
	s := &mockSession{}
	// Build an interaction where a different option is focused.
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommandAutocomplete,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "soundcron",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{
						Name:    "cron",
						Type:    discordgo.ApplicationCommandOptionString,
						Value:   "* * * * *",
						Focused: true,
					},
				},
			},
		},
	}
	handler.HandleTimezoneAutocomplete(s, i)

	if s.responded {
		t.Error("expected no response when focused option is not 'timezone'")
	}
}

func TestHandleTimezoneAutocomplete_NoBadEntries(t *testing.T) {
	// Verify that internal tzdata artifacts are not surfaced as choices.
	bad := []string{"+VERSION", "leap-seconds.list", "posixrules"}
	for _, prefix := range bad {
		s := &mockSession{}
		handler.HandleTimezoneAutocomplete(s, makeAutocompleteInteraction(prefix))
		names := choiceNames(s.response)
		if len(names) != 0 {
			t.Errorf("expected no choices for %q, got: %v", prefix, names)
		}
	}
}
