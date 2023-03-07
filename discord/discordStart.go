package discord

import (
	"context"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

// StartServer starts the Discord server
func StartServer() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		log.Fatal().Msgf("Error creating Discord session: %v", err)
	}

	dg.AddHandler(handler)

	err = dg.Open()
	if err != nil {
		log.Fatal().Msgf("Error opening Discord session: %v", err)
	}

	log.Debug().Msg("Discord bot started!")
	<-ctx.Done()
}
