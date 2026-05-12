package main

import (
	"context"
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
	"gorm.io/gorm"
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

		user, err := gorm.G[User](DB).Where("l_id = ?", LID).First(ctx)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleNewUser(ctx, v, LID, phoneNumber)
			return
		}

		handleExistingUser(ctx, v, user)
	}
}

func handleNewUser(ctx context.Context, evt *events.Message, LID, phoneNumber string) {
	contextInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	message := contextInfo.GetQuotedMessage().String()

	userCreationErr := gorm.G[User](DB).Create(ctx,
		&User{
			LID: LID, DisplayName: evt.Info.PushName, PhoneNumber: phoneNumber,
			Messages: []Message{{
				StanzaID: contextInfo.GetStanzaID(), SentAt: evt.Info.Timestamp, Type: MessageTypeText,
				MessageAttachments: []MessageAttachment{{Body: &message}},
			}},
		})

	if userCreationErr != nil {
		_, sendErr := client.SendMessage(
			context.Background(), evt.Info.Sender, &waE2E.Message{
				Conversation: proto.String(
					ServerUnknownErrMsg),
			})
		slog.Error("User was not created and an Error message send to user")
		if sendErr != nil {
			slog.Error(ErrMsgNotSend)
		}
		return
	}

	_, sendErr := client.SendMessage(
		context.Background(), evt.Info.Sender, &waE2E.Message{
			Conversation: proto.String(
				WelcomeMsg),
		})
	if sendErr != nil {
		slog.Error(ErrMsgNotSend)
	}
}

func handleExistingUser(ctx context.Context, evt *events.Message, user User) {
	contextInfo := evt.Message.GetExtendedTextMessage().GetContextInfo()
	message := contextInfo.GetQuotedMessage().String()

	msgCreationErr := gorm.G[Message](DB).Create(ctx,
		&Message{
			UserID: user.ID, StanzaID: contextInfo.GetStanzaID(), SentAt: evt.Info.Timestamp, Type: MessageTypeText,
			MessageAttachments: []MessageAttachment{{Body: &message}},
		})

	if msgCreationErr != nil {
		_, sendErr := client.SendMessage(
			context.Background(), evt.Info.Sender, &waE2E.Message{
				Conversation: proto.String(
					ServerUnknownErrMsg),
			})
		slog.Error("Messsage was not created and an Error message send to user")
		if sendErr != nil {
			slog.Error(ErrMsgNotSend)
		}
	}
}

// getLIDFromEvent returns LID and PhoneNumber from the event
// whatsapp is migrating from JID to LID so it is important to
// store LID
func getLIDAndNumberFromEvent(evt *events.Message) (string, string) {
	if strings.Contains(evt.Info.Sender.String(), "@lid") {
		return evt.Info.Sender.User, evt.Info.SenderAlt.User
	}
	// TODO: This block is to be testd as I don't have number that send JID
	return evt.Info.SenderAlt.String(), evt.Info.Sender.User
}

var client *whatsmeow.Client

const (
	ServerUnknownErrMsg = "An unknown Error occured, please try after sometime"
	ErrMsgNotSend       = "The error message was not send to the user"
	WelcomeMsg          = "Hi Ms.Muneera, I am your welcome bot. Nice to meet you !!"
)

func startWaServerAndListenEvt() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:bot.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
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

	// Listen to Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func configureSlog() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})))
}
