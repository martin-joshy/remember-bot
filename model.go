package main

import (
	"context"
	"database/sql"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
	"remember-bot/db"
)

//go:embed sql/schema.sql
var ddl string

type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeDocument MessageType = "document"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
)

var queries *db.Queries

func setupDB() {
	database, err := sql.Open("sqlite3", "bot.db?_foreign_keys=on")
	if err != nil {
		panic("failed to connect database")
	}

	if _, err := database.ExecContext(context.Background(), ddl); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	queries = db.New(database)
}
