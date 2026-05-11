package main

import (
	"context"
	"errors"
	"fmt"
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

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())

		contextInfo := v.Message.GetExtendedTextMessage().GetContextInfo()

		if contextInfo != nil {
			quotedID := contextInfo.GetStanzaID()
			quotedMessage := contextInfo.GetQuotedMessage()

			fmt.Println("Id : ", quotedID, ", Message : ", quotedMessage)

		}

		msgUserInfo := v.Info
		fmt.Println("PhoneNumber :", msgUserInfo.SenderAlt.User, "DisplayName : ", msgUserInfo.PushName, "JID : ", msgUserInfo.Sender.User)
		LID, phoneNumber := getLIDAndNumberFromEvent(v)
		message := contextInfo.GetQuotedMessage().String()
		// listen to only the whitelisted jid as we dont want
		// no else to use

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// check if the jid exist in the db
		user, err := gorm.G[User](DB).Where("l_id = ?", LID).First(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err := gorm.G[User](DB).Create(ctx, &User{
				LID: LID, DisplayName: msgUserInfo.PushName, PhoneNumber: phoneNumber,
				Messages: []Message{{
					StanzaID: contextInfo.GetStanzaID(), SentAt: msgUserInfo.Timestamp, Type: MessageTypeText,
					MessageAttachments: []MessageAttachment{{Body: &message}},
				}},
			})
			serverUnknownErrMsg := "An unknown Error occured, please try after sometime"
			if err != nil {
				_, sendErr := client.SendMessage(context.Background(), msgUserInfo.Sender, &waE2E.Message{
					Conversation: proto.String(serverUnknownErrMsg),
				})
			}
			// sendWelcomeText()
			_, sendErr := client.SendMessage(context.Background(), msgUserInfo.Sender, &waE2E.Message{
				Conversation: proto.String("Hello, World!"),
			})

			if sendErr != nil {
				fmt.Println(sendErr)
			}
		} else {
			err := gorm.G[Message](DB).Create(ctx, &Message{
				UserID: user.ID, StanzaID: contextInfo.GetStanzaID(), SentAt: msgUserInfo.Timestamp, Type: MessageTypeText,
				MessageAttachments: []MessageAttachment{{Body: &message}},
			})
			fmt.Println(err)
		}
		_, sendErr := client.SendMessage(context.Background(), msgUserInfo.Sender, &waE2E.Message{
			Conversation: proto.String("Hello, World!"),
		})

		if sendErr != nil {
			fmt.Println(sendErr)
		}
		// and send welcome
		// then storeForwards() or else only storeForwards()
		// send error or success for storeForwards()
	}
}

// getLIDFromEvent returns LID and PhoneNumber from the event
// whatsapp is migrating from JID to LID so it is important to
// store LID
func getLIDAndNumberFromEvent(evt *events.Message) (string, string) {
	if strings.Contains(evt.Info.Sender.String(), "@lid") {
		return evt.Info.Sender.User, evt.Info.SenderAlt.User
	} else {
		// TODO: This block is to be testd as I don't have number that send JID
		return evt.Info.SenderAlt.String(), evt.Info.Sender.User
	}
}

// func sendWelcomeText() string { // welcome text for new user
// 	panic("Not implimented")
// }

// func storeMsgForwards(any) error { // think if I should use generics
// 	// this function should handle all the Forward type case
// 	// string
// 	// store the messages with type text and the attachment with body
// 	// and with tag dump as the default tag
// 	panic("Not implimented")
// }

var client *whatsmeow.Client

func main() {
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
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
