package handler_test

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/handler"
)

func TestCommandToAddFileRequest(t *testing.T) {

	emptyOptions := []*discordgo.ApplicationCommandInteractionDataOption{}

	tc := []struct {
		name        string
		attachments map[string]*discordgo.MessageAttachment
		options     []*discordgo.ApplicationCommandInteractionDataOption
		expected    *handler.SoundCronAddFileRequest
		err         bool
	}{
		{
			name:        "Command with no attachments should return error",
			attachments: map[string]*discordgo.MessageAttachment{},
			options:     emptyOptions,
			expected:    nil,
			err:         true,
		},
		{
			name: "Command with multiple attachments should return error",
			attachments: map[string]*discordgo.MessageAttachment{
				"attachment1": {ID: "attachment1"},
				"attachment2": {ID: "attachment2"},
			},
			options:  emptyOptions,
			expected: nil,
			err:      true,
		},
	}

	for _, testCase := range tc {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := handler.CommandToAddFileRequest(testCase.attachments, testCase.options)
			if testCase.err {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Errorf("expected non-nil result but got nil")
				} else if result.Attachment.ID != testCase.expected.Attachment.ID {
					t.Errorf("expected attachment ID %s, got %s", testCase.expected.Attachment.ID, result.Attachment.ID)
				}
			}
		})
	}

}
