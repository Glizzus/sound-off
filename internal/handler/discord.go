package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"

	"log/slog"

	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/glizzus/sound-off/internal/schedule"
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

func CheckSoundCronAlreadyExists(candidate repository.SoundCron, soundCrons []repository.SoundCron) error {
	for _, soundCron := range soundCrons {
		if soundCron.Name == candidate.Name && soundCron.GuildID == candidate.GuildID {
			return &SoundCronAlreadyExistsError{
				GuildID: candidate.GuildID,
				Name:    candidate.Name,
			}
		}
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

func DoListSoundCrons(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	lister repository.SoundCronLister,
) error {
	ctx := context.Background()

	soundCrons, err := lister.List(ctx, i.GuildID)
	if err != nil {
		return fmt.Errorf("failed to list soundcrons: %w", err)
	}
	if len(soundCrons) == 0 {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No soundcrons found",
			},
		})
	}
	var responseContent string
	for _, sc := range soundCrons {
		responseContent += fmt.Sprintf("Name: %s, ID: %s\n", sc.Name, sc.ID)
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseContent,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to respond to interaction: %w", err)
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

	addFileHandler := AddFileHandler{
		Repo:          repo,
		AudioPiper:    audioPiper,
		UUIDGenerator: uuidGenerator,
	}

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
			case "list":
				err := DoListSoundCrons(s, i, repo)
				if err != nil {
					slog.Warn("Failed to list soundcrons", "error", err)
					return
				}
			case "add":
				if len(subCommand.Options) == 0 {
					slog.Warn("No subcommand provided for soundcron add command")
					return
				}
				subCommandGroup := subCommand.Options[0]
				switch subCommandGroup.Name {
				case "file":
					addFileRequest, err := CommandToAddFileRequest(
						command.Resolved.Attachments,
						subCommandGroup.Options,
					)
					if err != nil {
						slog.Warn("Failed to parse add file request", "error", err)
						err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: "Invalid request format",
							},
						})
						if err != nil {
							slog.Error("Failed to respond to interaction", "error", err)
						}
					}

					err = addFileHandler.Handle(
						i.GuildID,
						addFileRequest,
					)
					if err != nil {
						errorMessage := "Internal server error - please try again later"
						var ue *UserError
						if errors.As(err, &ue) {
							errorMessage = ue.Message
						} else {
							slog.Error("Failed to handle add file request", "error", err)
						}
						err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &discordgo.InteractionResponseData{
								Content: errorMessage,
								Flags:   discordgo.MessageFlagsEphemeral,
							},
						})
						if err != nil {
							slog.Error("Failed to respond to interaction", "error", err)
						}
					}
				}
			}
		}
	}
}

type AddFileHandler struct {
	Repo          repository.SoundCronRepository
	AudioPiper    *AudioPiper
	UUIDGenerator generator.UUIDGenerator
}

func (h *AddFileHandler) Handle(
	guildID string,
	addFileRequest *SoundCronAddFileRequest,
) error {
	id, err := h.UUIDGenerator.Next()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	soundCron := repository.SoundCron{
		ID:       id,
		Name:     addFileRequest.Name,
		GuildID:  guildID,
		Cron:     addFileRequest.Cron,
		FileSize: int64(addFileRequest.Attachment.Size),
	}

	ctx := context.Background()

	soundCrons, err := h.Repo.List(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to list soundcrons: %w", err)
	}

	err = CheckStorageAvailable(soundCrons, soundCron.FileSize, MaxStorageSize)
	if err != nil {
		return &UserError{
			Message: "Storage limit exceeded",
		}
	}

	err = CheckSoundCronAlreadyExists(soundCron, soundCrons)
	if err != nil {
		return &UserError{
			Message: "Soundcron with this name already exists",
		}
	}

	err = schedule.ValidateCron(soundCron.Cron)
	if err != nil {
		return &UserError{
			Message: "Invalid cron expression",
		}
	}

	err = h.Repo.Save(ctx, soundCron)
	if err != nil {
		return fmt.Errorf("failed to pipe audio: %w", err)
	}

	err = h.Repo.Save(ctx, soundCron)
	if err != nil {
		return fmt.Errorf("failed to save soundcron: %w", err)
	}

	return nil
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
