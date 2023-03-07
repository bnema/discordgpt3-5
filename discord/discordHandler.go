package discord

import (
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bnema/discordgpt3-5/db"
	"github.com/bnema/discordgpt3-5/openai"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

// simulateTyping simulates bot typing while the bot is generating a response
func simulateTyping(s *discordgo.Session, channelID string) {
	typingInterval := time.Duration(rand.Intn(2)+1) * time.Second // set a random typing interval between 1-3 seconds
	pauseInterval := time.Duration(rand.Intn(2)+1) * time.Second  // set a random pause interval between 1-3 seconds
	maxDuration := time.Duration(rand.Intn(20)+5) * time.Second   // set a random max duration between 5-25 seconds
	loopQuantity := rand.Intn(10) + 1                             // set a random loop quantity between 1-10
	// Start with typing
	s.ChannelTyping(channelID)
	// Pause for a random amount of time
	time.Sleep(typingInterval)
	// Loop a random amount of times
	for i := 0; i < loopQuantity; i++ {
		// Pause for a random amount of time
		time.Sleep(pauseInterval)
		// Check if max duration has been reached
		if pauseInterval > maxDuration {
			break
		}
		// Start with typing
		s.ChannelTyping(channelID)
	}
}

// handler handles the discord messages
func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If message contains only an emoji or a simple word, ignore it BUT not if "JP", "jp", "Jp", "jP",
	if !strings.Contains(m.Content, "JP") && !strings.Contains(m.Content, "jp") && !strings.Contains(m.Content, "Jp") && !strings.Contains(m.Content, "jP") && len(m.Content) < 3 {
		return
	}

	// if command is "!systemprompt" then create a new system prompt with the next message
	if strings.Contains(m.Content, "!systemprompt") {
		// Add the rest of the message as the system prompt
		systemPrompt := strings.Replace(m.Content, "!systemprompt", "", 1)
		err := openai.CreateNewSystemPrompt(systemPrompt)
		if err != nil {
			log.Error().Msgf("unable to create new system prompt: %v", err)
		} else {
			// send a message to the channel + send the new system prompt to the chatgpt
			_, err := s.ChannelMessageSend(m.ChannelID, "New system prompt created")
			// send the new system prompt to the chatgpt
			openai.SendToChatGPT(m.ChannelID, "system", systemPrompt)

			if err != nil {
				log.Error().Msgf("unable to send message to discord: %v", err)
			}
		}
		return
	}

	// if channelID is not "DISCORD_CHANNEL_ID"then ignore it
	if m.ChannelID != os.Getenv("DISCORD_CHANNEL_ID") {
		return
	}

	// resetdb command to reset the database
	if m.Content == "!resetdb" {
		// reset the database
		err := db.ResetDatabase()
		if err != nil {
			log.Error().Msgf("unable to reset the database: %v", err)
		} else {
			// send a message to the channel
			_, err := s.ChannelMessageSend(m.ChannelID, "Database reset")
			if err != nil {
				log.Error().Msgf("unable to send message to discord: %v", err)
			}
			// Set an empty system prompt
			err = openai.CreateNewSystemPrompt("")
			if err != nil {
				log.Error().Msgf("unable to create new system prompt: %v", err)
			}
		}
		return
	}

	// Outgoing message to chatgpt
	outgoingMsg := m.Content
	chatId := m.ChannelID
	userName := m.Author.Username
	log.Debug().Msg(outgoingMsg)

	// Use is typing while the bot is thinking
	simulateTyping(s, m.ChannelID)
	chatResp := openai.SendToChatGPT(chatId, userName, outgoingMsg)
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

		// Regex to remove the username: at the beginning of the message (also catch if user has a space in their name)
		re := regexp.MustCompile(`^.*?: `)
		incomingMsg.Content = re.ReplaceAllString(incomingMsg.Content, "")
		// Send a message to the channel
		s.ChannelMessageSend(m.ChannelID, incomingMsg.Content)
	}
}
