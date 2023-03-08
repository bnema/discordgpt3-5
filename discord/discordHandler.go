package discord

import (
	"fmt"
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

// Define a struct to hold the message information
type messageInfo struct {
	session    *discordgo.Session
	message    *discordgo.MessageCreate
	typingTime time.Duration
}

type FIFO struct {
	queue []string
}

// Create a new FIFO queue
var fifoQueue = FIFO{}

// Add a new element to the queue
func (f *FIFO) Enqueue(element string) {
	f.queue = append(f.queue, element)
}

// Remove the first element from the queue
func (f *FIFO) Dequeue() string {
	if len(f.queue) == 0 {
		return ""
	}
	element := f.queue[0]
	f.queue = f.queue[1:len(f.queue)]
	return element
}

// Return the first element from the queue
func (f *FIFO) IsEmpty() bool {
	return len(f.queue) == 0
}

// simulateTyping simulates bot typing while the bot is generating a response
func simulateTyping(channel chan messageInfo) {
	for {
		// Wait for a message to be received
		msgInfo := <-channel
		s := msgInfo.session
		m := msgInfo.message
		typingInterval := msgInfo.typingTime

		// Start with typing
		s.ChannelTyping(m.ChannelID)
		// Pause for a random amount of time
		time.Sleep(typingInterval)
		// Loop a random amount of times
		for i := 0; i < rand.Intn(10)+1; i++ {
			// Pause for a random amount of time
			time.Sleep(time.Duration(rand.Intn(2)+1) * time.Second)
			// Check if max duration has been reached
			if typingInterval > (time.Duration(rand.Intn(20)+5) * time.Second) {
				break
			}
			// Start with typing
			s.ChannelTyping(m.ChannelID)
		}
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
	typingInterval := time.Duration(rand.Intn(5)+5) * time.Second
	// Create a channel to send the message information
	channel := make(chan messageInfo)
	// Start the simulateTyping function
	go simulateTyping(channel)
	// Send the message information to the channel
	channel <- messageInfo{
		session:    s,
		message:    m,
		typingTime: typingInterval,
	}

	// Send the message to chatgpt with the queue number
	chatResp := openai.SendToChatGPT(chatId, userName, outgoingMsg)

	// Get the response from chatgpt with the queue number
	for _, choice := range chatResp {
		incomingMsg := choice.Message
		log.Printf("role=%q, content=%q", incomingMsg.Role, incomingMsg.Content)

		// Regex to remove the username: at the beginning of the message (also catch if user has a space in their name)
		re := regexp.MustCompile(`^.*?: `)
		incomingMsg.Content = re.ReplaceAllString(incomingMsg.Content, "")

		// Add to FIFO queue
		fifoQueue.Enqueue(incomingMsg.Content)
		fmt.Println(fifoQueue)

		// Traitement des éléments dans l'ordre d'arrivée
		for !fifoQueue.IsEmpty() {
			// Get the first element of the queue
			firsMsg := fifoQueue.Dequeue()
			// Send a message to the channel with the response
			_, err := s.ChannelMessageSend(m.ChannelID, firsMsg)
			//  if error "code": 50035 then split the message in 2 and send it
			if err != nil && err.(*discordgo.RESTError).Message.Code == 50035 {
				// Split the message in 2
				splitMessage := strings.Split(incomingMsg.Content, " ")
				// Send the first part of the message
				_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(splitMessage[:len(splitMessage)/2], " "))
				if err != nil {
					log.Error().Msgf("unable to send message to discord: %v", err)
				}
				// Send the second part of the message
				_, err = s.ChannelMessageSend(m.ChannelID, strings.Join(splitMessage[len(splitMessage)/2:], " "))
				if err != nil {
					log.Error().Msgf("unable to send message to discord: %v", err)
				}
			} else if err != nil {
				log.Error().Msgf("unable to send message to discord: %v", err)
			}
		}
	}
}
