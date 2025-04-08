package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"

	"log/slog"

	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/glizzus/sound-off/internal/util"
)

type ReadyHandler = func(*discordgo.Session, *discordgo.Ready)
type InteractionCreateHandler = func(*discordgo.Session, *discordgo.InteractionCreate)

var ReadyLog = func(s *discordgo.Session, r *discordgo.Ready) {
	username := r.User.Username
	userID := r.User.ID
	slog.Info("Bot is ready", "username", username, "userID", userID)
}

type SoundCronAddFileRequest struct {
	Attachment *discordgo.MessageAttachment
	Cron       string
	Name       string
}

func CommandToAddFileRequest(
	attachments map[string]*discordgo.MessageAttachment,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) (*SoundCronAddFileRequest, error) {
	attachment, err := util.GetOne(attachments)
	if err != nil {
		return nil, err
	}

	var cron string
	var name string

	for _, option := range options {
		switch option.Name {
		case "cron":
			if option.Type != discordgo.ApplicationCommandOptionString {
				return nil, fmt.Errorf("invalid type for cron option")
			}
			cron = option.StringValue()
		case "name":
			if option.Type != discordgo.ApplicationCommandOptionString {
				return nil, fmt.Errorf("invalid type for name option")
			}
			name = option.StringValue()
		}
	}

	if cron == "" {
		return nil, fmt.Errorf("cron option is required")
	}
	if name == "" {
		name = attachment.Filename
	}

	return &SoundCronAddFileRequest{
		Attachment: attachment,
		Cron:       cron,
		Name:       name,
	}, nil
}

const MaxStorageSize = 10 * 1024 * 1024 // 10 MB

type StorageLimitError struct {
	Requested int64
	Current   int64
	Max       int64
}

func (e *StorageLimitError) Error() string {
	return fmt.Sprintf("storage limit exceeded: requested %d, current %d, max %d", e.Requested, e.Current, e.Max)
}

var _ error = (*StorageLimitError)(nil)

func CheckStorageAvailable(soundCrons []repository.SoundCron, requested, maxStorage int64) error {
	var totalSize int64
	for _, soundCron := range soundCrons {
		totalSize += soundCron.FileSize
	}

	if totalSize+requested > maxStorage {
		return &StorageLimitError{
			Requested: requested,
			Current:   totalSize,
			Max:       maxStorage,
		}
	}
	return nil
}

func CheckSoundCronAlreadyExists(soundCrons []repository.SoundCron) error {
	type uniqueConstraint struct {
		GuildID string
		Name    string
	}

	uniqueMap := make(map[uniqueConstraint]struct{})
	for _, soundCron := range soundCrons {
		key := uniqueConstraint{
			GuildID: soundCron.GuildID,
			Name:    soundCron.Name,
		}
		if _, exists := uniqueMap[key]; exists {
			return fmt.Errorf("soundcron already exists for guild %s with name %s", soundCron.GuildID, soundCron.Name)
		}
		uniqueMap[key] = struct{}{}
	}
	return nil
}

// HTTPClient is an abstraction for making HTTP requests.
// The implementation is usually Go's stdlib http.Client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AudioPiper is a struct that performs the operation
// of downloading and immediately uploading.
type AudioPiper struct {
	blobStorage datalayer.BlobStorage
	httpClient  HTTPClient
	prefix      string
}

func (a *AudioPiper) Pipe(ctx context.Context, key, sourceURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	log.Printf("Downloading file from %s", sourceURL)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response with status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	key = a.prefix + "/" + key
	err = a.blobStorage.Put(ctx, key, resp.Body, datalayer.PutOptions{
		Size:        resp.ContentLength,
		ContentType: resp.Header.Get("Content-Type"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func MakeInteractionCreateHandler(
	repo *repository.PostgresSoundCronRepository,
	blobStorage datalayer.BlobStorage,
) InteractionCreateHandler {

	audioPiper := &AudioPiper{
		blobStorage: blobStorage,
		httpClient:  http.DefaultClient,
	}

	uuidGenerator := generator.UUIDGenerator{}

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		command := i.ApplicationCommandData()
		switch command.Name {
		case "ping":
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Pong!",
				},
			})
			if err != nil {
				slog.Error("Failed to respond to ping command", "error", err)
			}
		case "soundcron":
			if len(command.Options) == 0 {
				slog.Warn("No options provided for soundcron command")
				return
			}
			subCommand := command.Options[0]
			switch subCommand.Name {
			case "add":
				if len(subCommand.Options) == 0 {
					slog.Warn("No subcommand provided for soundcron add command")
					return
				}
				subCommandGroup := subCommand.Options[0]
				switch subCommandGroup.Name {
				case "file":
					addFileRequest, err := CommandToAddFileRequest(command.Resolved.Attachments, subCommandGroup.Options)
					if err != nil {
						slog.Warn("Failed to convert command to add file request", "error", err)
						return
					}

					id, err := uuidGenerator.Next()
					if err != nil {
						slog.Warn("Failed to generate UUID", "error", err)
						return
					}

					soundCron := repository.SoundCron{
						ID:       id,
						Name:     addFileRequest.Name,
						GuildID:  i.GuildID,
						Cron:     addFileRequest.Cron,
						FileSize: int64(addFileRequest.Attachment.Size),
					}

					ctx := context.Background()
					soundCrons, err := repo.List(ctx, i.GuildID)
					if err != nil {
						slog.Warn("Failed to list soundcrons", "error", err)
						return
					}

					err = CheckStorageAvailable(soundCrons, soundCron.FileSize, MaxStorageSize)
					if err != nil {
						slog.Warn("Storage limit exceeded", "error", err)
						return
					}

					err = CheckSoundCronAlreadyExists(soundCrons)
					if err != nil {
						slog.Warn("Soundcron already exists", "error", err)
						return
					}

					log.Printf("About to pipe audio: %s", addFileRequest.Attachment.ProxyURL)
					err = audioPiper.Pipe(ctx, id, addFileRequest.Attachment.ProxyURL)
					if err != nil {
						slog.Warn("Failed to pipe audio", "error", err)
						return
					}

					err = repo.Save(ctx, soundCron)
					if err != nil {
						slog.Warn("Failed to save soundcron", "error", err)
						return
					}
				}
			}
		}
	}
}

type Handlers struct {
	Ready             ReadyHandler
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
