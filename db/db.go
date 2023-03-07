package db

import (
	"github.com/rs/zerolog/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbFile = "database/chats.db"
	DB     *gorm.DB
)

type Message struct {
	gorm.Model
	ID       uint   `gorm:"primaryKey" json:"id"`
	ChatID   string `json:"chatId,omitempty"`   // Discord ChannelID
	Role     string `json:"role,omitempty"`     // chatgpt rol
	UserName string `json:"userName,omitempty"` // Discord UserName
	Content  string `json:"content,omitempty"`  // message content

	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type SystemPrompt struct {
	gorm.Model
	Prompt string `json:"prompt,omitempty"`
}

// ConnectDB
func ConnectDB() error {
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger: logger.Default,
	})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Message{}, &SystemPrompt{})

	DB = db
	log.Debug().Msg("database migrated")
	return nil
}

// FindMessages finds the prevous users conversations from the telegrams conversation id
func FindMessages(chatId string) ([]Message, error) {
	var messages []Message

	err := DB.Where(&Message{
		ChatID: chatId,
	}).Find(&messages).Error

	if err != nil {
		return nil, err
	}
	return messages, nil
}

// CreateMessage creates a new chat
func CreateMessage(msg Message) (*Message, error) {
	if err := DB.Create(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// DeleteMessage deletes a chat by id
func DeleteMessage(id uint) error {
	if err := DB.Where("id = ?", id).Delete(&Message{}).Error; err != nil {
		return err
	}
	return nil
}

func GetSystemPrompt() (string, error) {
	var systemPrompt SystemPrompt
	err := DB.First(&systemPrompt).Error
	if err != nil {
		return "", err
	}
	return systemPrompt.Prompt, nil
}

func CreateSystemPrompt(systemPrompt SystemPrompt) (*SystemPrompt, error) {
	// Check if there is already a system prompt
	var systemPromptExists SystemPrompt
	err := DB.First(&systemPromptExists).Error
	if err == nil {
		// update the system prompt
		if err := DB.Model(&systemPromptExists).Updates(&systemPrompt).Error; err != nil {
			return nil, err
		}
		return &systemPromptExists, nil
	} else if err != nil && err.Error() != "record not found" {
		return nil, err
	}

	// create a new system prompt
	if err := DB.Create(&systemPrompt).Error; err != nil {
		return nil, err
	}
	return &systemPrompt, nil
}

func ResetDatabase() error {
	// Truncate the database tables
	if err := DB.Migrator().DropTable(&Message{}, &SystemPrompt{}); err != nil {
		return err
	}
	if err := DB.AutoMigrate(&Message{}, &SystemPrompt{}); err != nil {
		return err
	}
	return nil
}
