package openai

import (
	"context"
	"os"

	"github.com/bnema/discordgpt3-5/db"
	"github.com/rakyll/openai-go"
	"github.com/rakyll/openai-go/chat"
	"github.com/rs/zerolog/log"
)

var (
	retainHistory bool
)

// SendToChatGPT send a message to chatgpt
func SendToChatGPT(chatId, userName string, textMsg string) []*chat.Choice {
	retainHistory = os.Getenv("RETAIN_HISTORY") == "true"
	var (
		ctx = context.Background()
		s   = openai.NewSession(os.Getenv("OPENAI_TOKEN"))

		// messages that will be sent to chatgpt (add userName before the textMsg)
		gptMsgs = make([]*chat.Message, 0)
	)

	// check if the user has a previous conversation
	prevMessages, err := db.FindMessages(chatId)
	if err != nil {
		log.Err(err)
	}

	// get the systems prompt model from the database
	prmptB, _ := db.GetSystemPrompt()

	// add system prompt if user is initially starting out the conversation
	if len(prevMessages) == 0 {
		// Say in discord "You need to setup a system prompt first"
		// add the system prompt to gpt
		gptMsgs = append(gptMsgs, &chat.Message{
			Role:    "system",
			Content: string(prmptB),
		})

	} else {
		// if we're retaining history
		if retainHistory {
			var historyMessagesLimit = 10
			// add the last 10 previous messages
			if len(prevMessages) > historyMessagesLimit {
				prevMessages = prevMessages[len(prevMessages)-historyMessagesLimit:]
			}
			// add the whole previous users conversation + current text message and send to chatgpt
			// this may include the previous prompt from the conversation
			for _, prevMsg := range prevMessages {
				gptMsgs = append(gptMsgs, &chat.Message{
					Role:    prevMsg.Role,
					Content: prevMsg.Content,
				})
			}
			// add the system prompt to gpt
			gptMsgs = append(gptMsgs, &chat.Message{
				Role:    "system",
				Content: string(prmptB),
			})
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
			_, err := db.CreateMessage(db.Message{
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
		_, err := db.CreateMessage(db.Message{
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
		_, err := db.CreateMessage(db.Message{
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

func CreateNewSystemPrompt(prompt string) error {
	// create a new system prompt
	_, err := db.CreateSystemPrompt(db.SystemPrompt{
		Prompt: prompt,
	})
	if err != nil {
		return err
	}
	return nil
}
