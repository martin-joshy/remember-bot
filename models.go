package main

import (
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
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	JID         string    `gorm:"column:jid;type:varchar(100);not null"`
	PhoneNumber string    `gorm:"column:phone_number;type:varchar(20);uniqueIndex"`
	DisplayName string    `gorm:"column:display_name;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	Tags        []Tag     `gorm:"foreignKey:UserID"`
	Messages    []Message `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "user"
}

type Message struct {
	ID          uint                `gorm:"primaryKey;autoIncrement"`
	UserID      uint                `gorm:"column:user_id;not null;index"`
	StanzaID    string              `gorm:"column:stanza_id;type:varchar(100)"`
	SentAt      time.Time           `gorm:"column:sent_at"`
	Type        MessageType         `gorm:"column:type;type:varchar(20);not null"`
	User        User                `gorm:"foreignKey:UserID"`
	Attachments []MessageAttachment `gorm:"foreignKey:MessageID"`
	Tags        []Tag               `gorm:"many2many:message_tag;"`
}

func (Message) TableName() string {
	return "message"
}

type MessageAttachment struct {
	ID        uint    `gorm:"primaryKey;autoIncrement"`
	MessageID uint    `gorm:"column:message_id;not null;index"`
	Body      string  `gorm:"column:body;type:text"`
	FilePath  string  `gorm:"column:file_path;type:varchar(512)"`
	FileName  string  `gorm:"column:file_name;type:varchar(255)"`
	MimeType  string  `gorm:"column:mime_type;type:varchar(30)"`
	FileSize  int     `gorm:"column:file_size"`
	Message   Message `gorm:"foreignKey:MessageID"`
}

func (MessageAttachment) TableName() string {
	return "message_attachment"
}

type Tag struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	UserID   uint      `gorm:"column:user_id;not null;index"`
	Name     string    `gorm:"column:name;type:varchar(100);not null"`
	User     User      `gorm:"foreignKey:UserID"`
	Messages []Message `gorm:"many2many:message_tag;"`
}

func (Tag) TableName() string {
	return "tag"
}

type MessageTag struct {
	MessageID uint `gorm:"primaryKey;column:message_id"`
	TagID     uint `gorm:"primaryKey;column:tag_id"`
}

func (MessageTag) TableName() string {
	return "message_tag"
}

var DB *gorm.DB

func init() {
	var err error
	DB, err = gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	DB.AutoMigrate(&User{}, &Message{}, &MessageAttachment{}, &Tag{}, &MessageTag{})
}
