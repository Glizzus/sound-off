package handler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/generator"
)

func InstanceIDFromInteraction(i *discordgo.InteractionCreate) string {
	var customID string

	switch i.Type {
	case discordgo.InteractionMessageComponent:
		customID = i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		customID = i.ModalSubmitData().CustomID
	default:
		return ""
	}

	return InstanceIDFromCustomID(customID)
}

func InstanceIDFromCustomID(customID string) string {
	parts := strings.SplitN(customID, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	return parts[1]
}

type FlowContext struct {
	InstanceID string
	State      map[string]any
}

type Node struct {
	ID      string
	Matcher func(*discordgo.InteractionCreate) bool
	Handler func(DiscordSession, *discordgo.InteractionCreate, *FlowContext) error
	Next    []*Node
}

type Flow struct {
	ID   string
	Root *Node
}

type session struct {
	flow *Flow
	node *Node
	ctx  *FlowContext
}

type FlowManager struct {
	flowsMu *sync.RWMutex
	flows   map[string]*Flow

	sessionsMu *sync.RWMutex
	sessions   map[string]*session

	idGenerator generator.Generator[string]
}

func NewFlowManager(idGenerator generator.Generator[string]) *FlowManager {
	if idGenerator == nil {
		idGenerator = &generator.UUIDV4Generator{}

	}
	return &FlowManager{
		flowsMu:     &sync.RWMutex{},
		flows:       make(map[string]*Flow),
		sessionsMu:  &sync.RWMutex{},
		sessions:    make(map[string]*session),
		idGenerator: idGenerator,
	}
}

func (fm *FlowManager) RegisterFlow(flow *Flow) {
	fm.flowsMu.Lock()
	defer fm.flowsMu.Unlock()

	if _, exists := fm.flows[flow.ID]; exists {
		panic("flow already registered")
	}
	fm.flows[flow.ID] = flow
}

func (fm *FlowManager) Router(s DiscordSession, i *discordgo.InteractionCreate) error {
	instanceID := InstanceIDFromInteraction(i)
	if instanceID != "" {
		fm.sessionsMu.RLock()
		session, inFlow := fm.sessions[instanceID]
		fm.sessionsMu.RUnlock()
		if inFlow {
			return fm.advance(s, i, session)
		}
	}

	return fm.initializeFlow(s, i)
}

func (fm *FlowManager) advance(
	s DiscordSession,
	i *discordgo.InteractionCreate,
	sess *session,
) error {
	finishFlow := func() {
		fm.sessionsMu.Lock()
		delete(fm.sessions, sess.ctx.InstanceID)
		fm.sessionsMu.Unlock()
	}

	if len(sess.node.Next) == 0 {
		finishFlow()
		return nil
	}

	var nextNode *Node
	for _, n := range sess.node.Next {
		if n.Matcher(i) {
			nextNode = n
			break
		}
	}
	if nextNode == nil {
		return nil
	}

	sess.node = nextNode
	if err := runHandler(s, i, sess); err != nil {
		return err
	}

	if len(nextNode.Next) == 0 {
		finishFlow()
	}
	return nil
}

var uuidGenerator = generator.UUIDV4Generator{}

func (fm *FlowManager) initializeFlow(s DiscordSession, i *discordgo.InteractionCreate) error {
	// Find the first matching flow
	var f *Flow
	for _, flow := range fm.flows {
		if flow.Root.Matcher(i) {
			f = flow
			break
		}
	}
	if f == nil {
		return nil
	}

	instanceID, err := fm.idGenerator.Next()
	if err != nil {
		return fmt.Errorf("failed to generate instance ID: %w", err)
	}

	ctx := &FlowContext{
		InstanceID: instanceID,
		State:      make(map[string]any),
	}
	newSess := &session{flow: f, node: f.Root, ctx: ctx}

	fm.sessionsMu.Lock()
	fm.sessions[instanceID] = newSess
	fm.sessionsMu.Unlock()

	return runHandler(s, i, newSess)
}

func runHandler(s DiscordSession, i *discordgo.InteractionCreate, sess *session) error {
	return sess.node.Handler(s, i, sess.ctx)
}
