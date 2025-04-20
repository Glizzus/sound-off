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
	"github.com/glizzus/sound-off/internal/presenters"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/glizzus/sound-off/internal/util"
)

const (
	ComponentIDIntervalSelect = "interval_select"
)

const (
	ModalIDCustomCronModal = "custom_cron_modal"
)

const (
	TextInputIDCronInput = "cron_input"
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

type DiscordSession interface {
	InteractionRespond(
		*discordgo.Interaction,
		*discordgo.InteractionResponse,
		...discordgo.RequestOption,
	) error

	InteractionResponseEdit(
		*discordgo.Interaction,
		*discordgo.WebhookEdit,
		...discordgo.RequestOption,
	) (*discordgo.Message, error)
}

var _ DiscordSession = (*discordgo.Session)(nil)

func DoListSoundCrons(
	s DiscordSession,
	i *discordgo.InteractionCreate,
	lister repository.SoundCronLister,
) error {
	ctx := context.Background()

	soundCrons, err := lister.List(ctx, i.GuildID)
	if err != nil {
		return fmt.Errorf("failed to list soundcrons: %w", err)
	}

	response := presenters.BuildListSoundCronsResponse(soundCrons)

	err = s.InteractionRespond(i.Interaction, response)
	if err != nil {
		return fmt.Errorf("failed to respond to interaction: %w", err)
	}
	return nil
}

var sessions = make(map[string]*SoundCronAddFileRequest)

type HandlerContext struct {
	Repo           *repository.PostgresSoundCronRepository
	AudioPiper     *AudioPiper
	UUIDGenerator  generator.UUIDV4Generator
	AddFileHandler *AddFileHandler
}

// NewDiscordInteractionHandler creates a new handler for Discord interactions.
// It uses the necessary types required by discordgo.
func NewDiscordInteractionHandler(
	repo *repository.PostgresSoundCronRepository,
	blobStorage datalayer.BlobStorage,
) func(*discordgo.Session, *discordgo.InteractionCreate) {
	internalHandler := NewInteractionHandler(repo, blobStorage)
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		internalHandler(s, i)
	}
}

func NewInteractionHandler(
	repo *repository.PostgresSoundCronRepository,
	blobStorage datalayer.BlobStorage,
) func(DiscordSession, *discordgo.InteractionCreate) {
	audioPiper := &AudioPiper{
		blobStorage: blobStorage,
		httpClient:  http.DefaultClient,
	}

	uuidGenerator := generator.UUIDV4Generator{}

	addFileHandler := &AddFileHandler{
		Repo:          repo,
		AudioPiper:    audioPiper,
		UUIDGenerator: uuidGenerator,
	}

	handlerCtx := &HandlerContext{
		Repo:           repo,
		AudioPiper:     audioPiper,
		UUIDGenerator:  uuidGenerator,
		AddFileHandler: addFileHandler,
	}

	return func(s DiscordSession, i *discordgo.InteractionCreate) {
		HandleInteractionCreate(handlerCtx, s, i)
	}
}

// HandleInteraction is the real handler for the interaction.
// discordgo uses reflection-based methods to call its handlers,
// which means we can not supply custom interfaces.
//
// Therefore, we perform all of our logic in this function
// and discordgo acts as a thin wrapper around this.
func HandleInteractionCreate(
	handlerCtx *HandlerContext,
	s DiscordSession,
	i *discordgo.InteractionCreate,
) {
	addFileHandler := handlerCtx.AddFileHandler
	repo := handlerCtx.Repo

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
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

					var userID string
					if i.Member != nil {
						userID = i.Member.User.ID
					} else {
						slog.Warn("No member information in interaction")
						return
					}

					var response *discordgo.InteractionResponse
					if addFileRequest.Cron == "" {
						sessions[userID] = addFileRequest

						menu := discordgo.SelectMenu{
							CustomID:    ComponentIDIntervalSelect,
							Placeholder: "Select an interval",
							Options: []discordgo.SelectMenuOption{
								{
									Label: "Every hour",
									Value: "@hourly",
								},
								{
									Label: "Cron (Custom)",
									Value: "cron",
								},
							},
						}
						row := discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{menu},
						}
						respData := discordgo.InteractionResponseData{
							Content:    "Choose an interval for your SoundCron:",
							Components: []discordgo.MessageComponent{row},
						}
						response = &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseChannelMessageWithSource,
							Data: &respData,
						}
					} else {
						err = addFileHandler.ProcessAddSoundCron(
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
							response = &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseChannelMessageWithSource,
								Data: &discordgo.InteractionResponseData{
									Content: errorMessage,
									Flags:   discordgo.MessageFlagsEphemeral,
								},
							}
						}
					}

					if response == nil {
						slog.Warn("discord response struct is nil")
					} else {
						err = s.InteractionRespond(i.Interaction, response)
						if err != nil {
							slog.Warn("failed to respond to add request", "error", err)
						}
					}
				}
			}
		}
	case discordgo.InteractionMessageComponent:
		component := i.MessageComponentData()
		switch component.CustomID {
		case ComponentIDIntervalSelect:
			modalData := discordgo.InteractionResponseData{
				CustomID: ModalIDCustomCronModal,
				Title:    "Enter Cron Expression",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							// TODO: min-max length
							discordgo.TextInput{
								CustomID: TextInputIDCronInput,
								Label:    "Cron Expression",
								Style:    discordgo.TextInputShort,
								Required: true,
							},
						},
					},
				},
			}
			response := &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &modalData,
			}
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				slog.Warn("failed to respond to component", "error", err)
			}
		}
	case discordgo.InteractionModalSubmit:
		modal := i.ModalSubmitData()
		switch modal.CustomID {
		case ModalIDCustomCronModal:
			components := modal.Components
			if len(components) == 0 {
				slog.Warn("no components found")
				return
			}
			// TODO: Prevent panics
			row := components[0].(*discordgo.ActionsRow)
			cronInput := row.Components[0].(*discordgo.TextInput)
			cronExpr := cronInput.Value

			userID := i.Member.User.ID

			addFileRequest := sessions[userID]
			addFileRequest.Cron = cronExpr
			addFileHandler.Handle(s, i, addFileRequest)
		}
	}

}

type AddFileHandler struct {
	Repo          repository.SoundCronRepository
	AudioPiper    *AudioPiper
	UUIDGenerator generator.UUIDV4Generator
}

var SoundCronAddDeferredResponse = &discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	Data: &discordgo.InteractionResponseData{
		Content: "Waiting!",
	},
}

var FinalResponseContent = "Done!"
var SoundCronAddFinalResponse = &discordgo.WebhookEdit{
	Content: &FinalResponseContent,
}

func (h *AddFileHandler) Handle(
	session DiscordSession,
	interaction *discordgo.InteractionCreate,
	addFileRequest *SoundCronAddFileRequest,
) {
	err := session.InteractionRespond(interaction.Interaction, SoundCronAddDeferredResponse)
	fmt.Println(err)

	err = h.ProcessAddSoundCron(interaction.GuildID, addFileRequest)
	fmt.Println(err)

	_, err = session.InteractionResponseEdit(interaction.Interaction, SoundCronAddFinalResponse)
	fmt.Println(err)
}

func (h *AddFileHandler) ProcessAddSoundCron(
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

	err = h.AudioPiper.Pipe(ctx, soundCron.ID, addFileRequest.Attachment.URL)
	if err != nil {
		return fmt.Errorf("failed to pipe audio: %w", err)
	}

	return nil
}

type Handlers struct {
	Ready             ReadyHandler
	InteractionCreate InteractionCreateHandler
}

const intents = discordgo.IntentGuilds | discordgo.IntentGuildMembers | discordgo.IntentGuildVoiceStates

func NewSession(token string, handlers Handlers) (*discordgo.Session, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error constructing discordgo session: %w", err)
	}

	s.AddHandler(handlers.Ready)
	s.AddHandler(handlers.InteractionCreate)

	s.Identify.Intents = intents
	return s, nil
}
