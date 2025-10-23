package main

import (
	"log"
	"path/filepath"

	"github.com/zelenin/go-tdlib/client"
)

func main() {
	// 初始化 TDLib 授权器
	authorizer := client.ClientAuthorizer(&client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join("/app/tdlib-db", "database"),
		FilesDirectory:      filepath.Join("/app/tdlib-db", "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               22067161,
		ApiHash:             "4318f60e99526473d075240b3de8a6ee",
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "0.1.0",
	})
	go client.CliInteractor(authorizer)

	tdlibClient, err := client.NewClient(authorizer)
	if err != nil {
		log.Fatal(err)
	}

	listener := tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch u := update.(type) {
		case *client.UpdateNewMessage:
			msg := u.Message

			// 获取群聊信息
			chat, err := tdlibClient.GetChat(&client.GetChatRequest{ChatId: msg.ChatId})
			if err != nil {
				log.Printf("New message from chat %d: (failed to get chat info: %v)\n", msg.ChatId, err)
				continue
			}

			// 提取消息文本（只处理 TextMessageContent 类型）
			text := ""
			if content, ok := msg.Content.(*client.MessageText); ok {
				text = content.Text.Text
			}

			log.Printf("New message from chat '%s' (ID %d): %s\n", chat.Title, chat.Id, text)
		}
	}
}
