package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"remember-bot/db"
)

func main() {
	configureSlog()
	setupDB()
	startWaServerAndListenEvt()
}

func eventHandler(evt any) {
	switch v := evt.(type) {
	case *events.Message:

		LID, phoneNumber := getLIDAndNumberFromEvent(v)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if strings.HasPrefix(v.Message.GetConversation(), "!") {
			if strings.HasPrefix(v.Message.GetConversation(), "!tag") {
				sendUserTaggedMsgs(ctx, v, LID)
			}
		}

		user, err := queries.GetUserByLID(ctx, LID)

		if errors.Is(err, sql.ErrNoRows) {
			handleNewUser(ctx, v, LID, phoneNumber)
			return
		}

		handleExistingUser(ctx, v, user)
	}
}

func sendUserTaggedMsgs(ctx context.Context, evt *events.Message, LID string) {
	parts := strings.SplitN(strings.TrimSpace(evt.Message.GetConversation()), " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		_, sendErr := client.SendMessage(ctx, evt.Info.Sender, &waE2E.Message{
			Conversation: proto.String("Usage: !tag <tag_name>"),
		})
		if sendErr != nil {
			slog.Error(ErrMsgNotSent)
		}
		return
	}
	tagName := strings.TrimSpace(parts[1])

	user, err := queries.GetUserByLID(ctx, LID)
	if err != nil {
		slog.Error("user not found", "lid", LID)
		return
	}
	_ = user
	_ = tagName
}

func ListMsgsByUserTag(ctx context.Context, userID int64, tagName string) ([]db.Message, error) {
	return queries.ListMessagesByUserTag(ctx, db.ListMessagesByUserTagParams{
		Name:   tagName,
		UserID: userID,
	})
}

func handleNewUser(ctx context.Context, evt *events.Message, LID, phoneNumber string) {
	contextInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	message := contextInfo.GetQuotedMessage().String()

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		LID:         LID,
		PhoneNumber: &phoneNumber,
		DisplayName: &evt.Info.PushName,
	})
	if err != nil {
		_, sendErr := client.SendMessage(
			ctx, evt.Info.Sender, &waE2E.Message{
				Conversation: proto.String(
					ServerUnknownErrMsg),
			})
		slog.Error("User was not created and an Error message send to user")
		if sendErr != nil {
			slog.Error(ErrMsgNotSent)
		}
		return
	}

	stanzaID := contextInfo.GetStanzaID()
	msg, msgErr := queries.CreateMessage(ctx, db.CreateMessageParams{
		UserID:   user.ID,
		StanzaID: &stanzaID,
		SentAt:   evt.Info.Timestamp,
		Type:     string(MessageTypeText),
	})

	if msgErr != nil {
		slog.Error("message was not created", "error", msgErr)
	} else {
		_, attErr := queries.CreateMessageAttachment(ctx, db.CreateMessageAttachmentParams{
			MessageID: msg.ID,
			Body:      &message,
		})
		if attErr != nil {
			slog.Error("failed to create message attachment", "error", attErr)
		}
	}

	_, sendErr := client.SendMessage(
		ctx, evt.Info.Sender, &waE2E.Message{
			Conversation: proto.String(
				WelcomeMsg),
		})
	if sendErr != nil {
		slog.Error(ErrMsgNotSent)
	}
}

func handleExistingUser(ctx context.Context, evt *events.Message, user db.User) {
	contextInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	message := contextInfo.GetQuotedMessage().String()

	stanzaID := contextInfo.GetStanzaID()
	msg, msgErr := queries.CreateMessage(ctx, db.CreateMessageParams{
		UserID:   user.ID,
		StanzaID: &stanzaID,
		SentAt:   evt.Info.Timestamp,
		Type:     string(MessageTypeText),
	})

	if msgErr != nil {
		_, sendErr := client.SendMessage(
			ctx, evt.Info.Sender, &waE2E.Message{
				Conversation: proto.String(
					ServerUnknownErrMsg),
			})
		slog.Error("messsage was not created and an error message send to user")
		if sendErr != nil {
			slog.Error(ErrMsgNotSent)
		}
		return
	}

	_, attErr := queries.CreateMessageAttachment(ctx, db.CreateMessageAttachmentParams{
		MessageID: msg.ID,
		Body:      &message,
	})
	if attErr != nil {
		slog.Error("failed to create message attachment", "error", attErr)
	}
}

func getLIDAndNumberFromEvent(evt *events.Message) (string, string) {
	if strings.Contains(evt.Info.Sender.String(), "@lid") {
		return evt.Info.Sender.User, evt.Info.SenderAlt.User
	}
	return evt.Info.SenderAlt.String(), evt.Info.Sender.User
}

var client *whatsmeow.Client

const (
	ServerUnknownErrMsg = "an unknown error occurred, please try again after some time"
	ErrMsgNotSent       = "the error message was not send to the user"
	WelcomeMsg          = "Hi Ms.Muneera, I am your welcome bot. Nice to meet you !!"
)

func startWaServerAndListenEvt() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:bot.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		qrChan, qrErr := client.GetQRChannel(context.Background())
		if qrErr != nil {
			slog.Error("Qr Error: ", "error", qrErr)
		}
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}
	defer client.Disconnect()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func configureSlog() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})))
}
