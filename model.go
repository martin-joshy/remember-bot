package main

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeDocument MessageType = "document"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
)

type User struct {
	ID          uint
	LID         string `gorm:"column:l_id;type:varchar(100);uniqueIndex;not null"`
	PhoneNumber string `gorm:"type:varchar(20);uniqueIndex"`
	DisplayName string `gorm:"type:text"`
	CreatedAt   time.Time
	Messages    []Message
}

type Message struct {
	ID                 uint
	UserID             uint   `gorm:"index;not null"`
	StanzaID           string `gorm:"type:varchar(100);index"`
	SentAt             time.Time
	Type               MessageType `gorm:"type:varchar(20);not null"`
	User               User
	Tags               []Tag `gorm:"many2many:message_tags;default:dump"`
	MessageAttachments []MessageAttachment
}

type MessageAttachment struct {
	ID        uint
	MessageID uint    `gorm:"index;not null"`
	Body      *string `gorm:"type:text"`
	FileName  string  `gorm:"type:varchar(255)"`
	FilePath  string  `gorm:"type:varchar(512)"`
	MimeType  string  `gorm:"type:varchar(255)"`
	FileSize  uint
	Message   Message
}

type Tag struct {
	ID       uint
	UserID   uint   `gorm:"not null;uniqueIndex:tag_user_name"`
	Name     string `gorm:"type:varchar(100);uniqueIndex:tag_user_name"`
	User     User
	Messages []Message `gorm:"many2many:message_tags"`
}

var DB *gorm.DB

func init() {
	fmt.Println("I am inside DBBB")
	var err error
	DB, err = gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = DB.AutoMigrate(&User{}, &Message{}, &MessageAttachment{}, &Tag{})
	if err != nil {
		panic("failed to migrate")
	}
}
