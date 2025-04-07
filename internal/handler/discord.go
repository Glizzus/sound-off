package handler

import (
	"github.com/bwmarrin/discordgo"

	"log/slog"
)

type ReadyHandler = func(*discordgo.Session, *discordgo.Ready)
type InteractionCreateHandler = func(*discordgo.Session, *discordgo.InteractionCreate)

var ReadyLog = func(s *discordgo.Session, r *discordgo.Ready) {
	username := r.User.Username
	userID := r.User.ID
	slog.Info("Bot is ready", "username", username, "userID", userID)
}

func MakeInteractionCreateHandler() InteractionCreateHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		slog.Info("Interaction created", "type", i.Type)
	}
}

type Handlers struct {
	Ready         ReadyHandler
	InteractionCreate InteractionCreateHandler
}

func NewSession(token string, handlers Handlers) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	s.AddHandler(handlers.Ready)
	s.AddHandler(handlers.InteractionCreate)

	return s, nil
}
