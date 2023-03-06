package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/rakyll/openai-go"
	"github.com/rakyll/openai-go/chat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	retainHistory bool
	promptName    = "prompt.txt"
)

func main() {
	// setup logger
	log.Logger = log.With().Caller().Logger()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Environment files
	err := godotenv.Load()
	if err != nil {
		log.Debug().Msg(err.Error())
	}

	retainHistory = os.Getenv("RETAIN_HISTORY") == "true"

	if err := ConnectDB(); err != nil {
		log.Fatal().Msg(err.Error())
	}

	// start server
	StartServer()
}

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

// SendToChatGPT send a message to chatgpt
func SendToChatGPT(chatId, userName string, textMsg string) []*chat.Choice {
	var (
		ctx = context.Background()
		s   = openai.NewSession(os.Getenv("OPENAI_TOKEN"))

		// messages that will be sent to chatgpt (add userName before the textMsg)
		gptMsgs = make([]*chat.Message, 0)
	)

	// check if the user has a previous conversation
	prevMessages, err := FindMessages(chatId)
	if err != nil {
		log.Err(err)
	}

	// get the systems prompt
	prmptB, _ := os.ReadFile(promptName)

	// limit the number of previous messages to avoid exceeding the token quota
	maxMessages := 20 // set a maximum number of messages to keep
	if len(prevMessages) > maxMessages {
		// remove the oldest messages
		prevMessages = prevMessages[len(prevMessages)-maxMessages:]
	}

	// add system prompt if user is initially starting out the conversation
	if len(prevMessages) == 0 {
		// create & add the systems prompt first
		log.Debug().Msg("added system prompt because its a first time user")
		gptMsgs = append(gptMsgs, &chat.Message{
			Role:    "user", // "system"
			Content: string(prmptB),
		})

	} else {
		// if we're retaining history
		if retainHistory {
			// add the whole previous users conversation + current text message and send to chatgpt
			// this may include the previous prompt from the conversation
			for _, prevMsg := range prevMessages {
				gptMsgs = append(gptMsgs, &chat.Message{
					Role:    prevMsg.Role,
					Content: prevMsg.Content,
				})
			}
		} else {
			// add only the system prompt to gpt
			gptMsgs = append(gptMsgs, &chat.Message{
				Role:    "user", // "system"
				Content: string(prmptB),
			})
		}
	}
	// add the current text message
	gptMsgs = append(gptMsgs, &chat.Message{
		Role:    "user",
		Content: userName + ": " + textMsg,
	})

	// process request
	client := chat.NewClient(s, "gpt-3.5-turbo-0301")
	resp, err := client.CreateCompletion(ctx, &chat.CreateCompletionParams{
		Messages: gptMsgs,
	})
	if err != nil {
		log.Error().Msgf("Failed to complete: %v", err)
		return nil
	}

	// save the new prompt + current text to DB
	if len(prevMessages) == 0 {
		for _, gptMsg := range gptMsgs {
			_, err := CreateMessage(Message{
				ChatID:   chatId,
				UserName: userName,
				Content:  gptMsg.Content,
				Role:     gptMsg.Role,

				// metrics for this single chat session
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			})
			if err != nil {
				log.Error().Msgf("unable to save message: %v", err)
			}
		}
	} else {
		// save the current content
		_, err := CreateMessage(Message{
			ChatID:   chatId,
			UserName: userName,
			Role:     "user",
			Content:  textMsg,

			// metrics for this single chat session
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
		if err != nil {
			log.Error().Msgf("unable to current message: %v", err)
		}
	}

	// save these reply responses
	for _, choice := range resp.Choices {
		_, err := CreateMessage(Message{
			ChatID:   chatId,
			UserName: userName,
			Role:     choice.Message.Role,
			Content:  choice.Message.Content,

			// metrics for this single chat session
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
		if err != nil {
			log.Error().Msgf("unable save chat response: %v", err)
		}
	}

	log.Info().
		Int("TotalTokens", resp.Usage.TotalTokens).
		Int("CompletionTokens", resp.Usage.CompletionTokens).
		Int("PromptTokens", resp.Usage.PromptTokens).
		Msg("usage")

	return resp.Choices
}

// handler handles the discord messages
func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If someone is citing someone else, ignore it
	if strings.Contains(m.Content, "@") {
		return
	}

	// if channelID is not "DISCORD_CHANNEL_ID"then ignore it
	if m.ChannelID != os.Getenv("DISCORD_CHANNEL_ID") {
		return
	}

	// Outgoing message to chatgpt
	outgoingMsg := m.Content
	chatId := m.ChannelID
	userName := m.Author.Username
	log.Debug().Msg(outgoingMsg)

	chatResp := SendToChatGPT(chatId, userName, outgoingMsg)
	if chatResp == nil {

		// Define an array of responses
		responses := []string{
			"Sorry, there seems to be a temporary issue. I'll keep trying and let you know as soon as it's back online.",
			"Hmmm, something's not quite right. I'm on the case and will update you when it's working again.",
			"Looks like I'm having a bit of a moment. I'm keeping an eye on it and will let you know when it's back up.",
			"Whoops, I seem to be down at the moment. I'll do my best to reconnect and keep you posted.",
			"That's bad, I can't seem to reach the destination endpoint. But I'll get back to you when I'm online.",
			"Oh no, I'm down. I'll keep trying and notify you when I'm back online.",
		}
		randIndex := rand.Intn(len(responses))

		// Send a message to the channel
		s.ChannelMessageSend(m.ChannelID, responses[randIndex])
		return
	}

	for _, choice := range chatResp {
		incomingMsg := choice.Message
		log.Printf("role=%q, content=%q", incomingMsg.Role, incomingMsg.Content)

		// Send a message to the channel
		s.ChannelMessageSend(m.ChannelID, incomingMsg.Content)

	}
}
