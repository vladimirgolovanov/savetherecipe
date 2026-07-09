package bot

import (
	"context"
	"fmt"
	"log"
	"savetherecipe/internal/handler"
	"savetherecipe/internal/models"
	"strings"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	captionLimit = 900
	startText    = "Отправьте первую ссылку, чтобы посмотреть, как это работает. \n Например, эту: https://www.instagram.com/reel/Cr0g43KIznu/?igshid=MTc4MmM1YmI2Ng%3D%3D"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	handler InputHandler
	imgDir  string
}

type InputHandler interface {
	Handle(ctx context.Context, text string) (*models.Result, error)
}

func New(api *tgbotapi.BotAPI, handler InputHandler) *Bot {
	return &Bot{
		api:     api,
		imgDir:  "img",
		handler: handler,
	}
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}
		go b.handleMessage(update.Message)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.sendStart(msg.Chat.ID)
	default:
		b.handleInput(msg)
	}
}

func (b *Bot) handleInput(msg *tgbotapi.Message) {
	ctx := context.Background()
	if msg.From != nil {
		ctx = handler.WithTelegramUserID(ctx, msg.From.ID)
	}

	result, err := b.handler.Handle(ctx, msg.Text)
	if err != nil {
		b.reportError(msg.Chat.ID, "handle input", err)
		return
	}
	if result == nil {
		return
	}

	if result.MessageType == models.MessageTypeInstagramURL {
		b.replacePost(msg, result)
	}
}

func (b *Bot) replacePost(msg *tgbotapi.Message, result *models.Result) {
	source := fmt.Sprintf("---\nИсточник: %s", result.SourceURL)

	if len([]rune(result.Caption)) <= captionLimit {
		b.sendPhotoWithCaption(msg.Chat.ID, result.ImageData, result.Caption+"\n"+source)
	} else {
		runes := []rune(result.Caption)
		firstPart := string(runes[:captionLimit])
		rest := strings.TrimSpace(string(runes[captionLimit:]))

		b.sendPhotoWithCaption(msg.Chat.ID, result.ImageData, firstPart)
		b.send(tgbotapi.NewMessage(msg.Chat.ID, rest+"\n"+source))
	}

	b.deleteMessage(msg.Chat.ID, msg.MessageID)
}

func (b *Bot) sendStart(chatID int64) {
	paths := []string{
		b.imgDir + "/1.jpg",
		b.imgDir + "/2.jpg",
		b.imgDir + "/3.jpg",
		b.imgDir + "/4.jpg",
	}

	var media []interface{}
	for i, path := range paths {
		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path))
		if i == 0 {
			photo.Caption = startText
		}
		media = append(media, photo)
	}

	group := tgbotapi.NewMediaGroup(chatID, media)
	if _, err := b.api.SendMediaGroup(group); err != nil {
		log.Printf("send start media group: %v", err)
		b.send(tgbotapi.NewMessage(chatID, startText))
	}
}

func (b *Bot) sendPhotoWithCaption(chatID int64, imgData []byte, caption string) {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  "photo.jpg",
		Bytes: imgData,
	})
	photo.Caption = caption
	b.send(photo)
}

func (b *Bot) deleteMessage(chatID int64, messageID int) {
	del := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := b.api.Request(del); err != nil {
		log.Printf("delete message: %v", err)
	}
}

func (b *Bot) send(c tgbotapi.Chattable) {
	if _, err := b.api.Send(c); err != nil {
		log.Printf("send: %v", err)
	}
}

func (b *Bot) reportError(chatID int64, action string, err error) {
	log.Printf("error %s: %v", action, err)
	sentry.CaptureException(fmt.Errorf("%s: %w", action, err))
	b.send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Не удалось %s. Попробуйте позже.", action)))
}
