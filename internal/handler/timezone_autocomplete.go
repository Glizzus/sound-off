package handler

// Regenerate timezones.txt with:
//
//go:generate sh -c "find /usr/share/zoneinfo -type f | sed 's|/usr/share/zoneinfo/||' | grep -E '^[A-Z]' | grep -vE '^(posixrules|leap-seconds\\.list|SECURITY|Factory|Leap)$' | sort -k1 > timezones.txt"

import (
	_ "embed"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/derekparker/trie"
)

//go:embed timezones.txt
var timezonesFile string

var commonTimezones = []string{
	"UTC",
	"America/New_York",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/Anchorage",
	"Pacific/Honolulu",
	"Europe/London",
	"Europe/Paris",
	"Europe/Berlin",
	"Europe/Moscow",
	"Asia/Dubai",
	"Asia/Kolkata",
	"Asia/Bangkok",
	"Asia/Shanghai",
	"Asia/Tokyo",
	"Asia/Seoul",
	"Australia/Sydney",
	"Pacific/Auckland",
	"America/Sao_Paulo",
	"America/Toronto",
	"America/Vancouver",
}

var timezoneTrie *trie.Trie

func init() {
	timezoneTrie = trie.New()
	for tz := range strings.SplitSeq(strings.TrimSpace(timezonesFile), "\n") {
		if tz != "" {
			timezoneTrie.Add(tz, nil)
		}
	}
}

// findFocusedOption searches the option tree recursively for the focused option.
func findFocusedOption(
	options []*discordgo.ApplicationCommandInteractionDataOption,
) *discordgo.ApplicationCommandInteractionDataOption {
	for _, opt := range options {
		if opt.Focused {
			return opt
		}
		if len(opt.Options) > 0 {
			if found := findFocusedOption(opt.Options); found != nil {
				return found
			}
		}
	}
	return nil
}

// HandleTimezoneAutocomplete responds to an autocomplete interaction for the timezone option.
// Empty input returns commonTimezones; non-empty input returns trie prefix-search results capped at 25.
func HandleTimezoneAutocomplete(s DiscordSession, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	focused := findFocusedOption(data.Options)
	if focused == nil || focused.Name != "timezone" {
		return
	}

	prefix := focused.StringValue()

	var matches []string
	if prefix == "" {
		matches = commonTimezones
	} else {
		results := timezoneTrie.PrefixSearch(prefix)
		if len(results) > 25 {
			results = results[:25]
		}
		matches = results
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(matches))
	for _, tz := range matches {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  tz,
			Value: tz,
		})
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		slog.Error("Failed to respond to timezone autocomplete", "error", err)
	}
}
