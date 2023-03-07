package main

import (
	"os"

	"github.com/bnema/discordgpt3-5/db"
	"github.com/bnema/discordgpt3-5/discord"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// setup logger
	log.Logger = log.With().Caller().Logger()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := db.ConnectDB(); err != nil {
		log.Fatal().Msg(err.Error())
	}

	// start server
	discord.StartServer()
}
