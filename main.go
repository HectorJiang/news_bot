package main

import (
	"log"

	"github.com/zelenin/go-tdlib/client"
)

func main() {
	tdlibClient, err := client.NewClient(client.Config{
		APIID:              "22067161",
		APIHash:            "4318f60e99526473d075240b3de8a6ee",
		SystemLanguageCode: "en",
		DeviceModel:        "Server",
		SystemVersion:      "1.0",
		ApplicationVersion: "0.1",
		UseTestDataCenter:  false,
		DatabaseDirectory:  "./tdlib-db",
		FileDirectory:      "./tdlib-files",
	})
	if err != nil {
		log.Fatal(err)
	}

	listener := tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch u := update.Data.(type) {
		case *client.UpdateNewMessage:
			msg := u.Message
			chat, err := tdlibClient.GetChat(msg.ChatId)
			if err != nil {
				log.Printf("New message from chat %d: %s (failed to get chat info: %v)\n", msg.ChatId, msg.Content, err)
				continue
			}

			log.Printf("New message from chat '%s' (ID %d): %s\n", chat.Title, chat.Id, msg.Content)
		}
	}
}
