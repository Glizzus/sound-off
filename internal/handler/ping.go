package handler

import "github.com/bwmarrin/discordgo"

var PingFlow = &Flow{
	ID: "ping",
	Root: &Node{
		ID: "ping",
		Matcher: func(i *discordgo.InteractionCreate) bool {
			if i.Type != discordgo.InteractionApplicationCommand {
				return false
			}
			return i.ApplicationCommandData().Name == "ping"
		},
		Handler: func(s DiscordSession, i *discordgo.InteractionCreate, ctx *FlowContext) error {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Pong!",
				},
			})
		},
	},
}
