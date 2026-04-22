package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"savetherecipe/internal/instagram"
)

const (
	captionLimit = 900
	startText    = "Отправьте первую ссылку, чтобы посмотреть, как это работает. \n Например, эту: https://www.instagram.com/reel/Cr0g43KIznu/?igshid=MTc4MmM1YmI2Ng%3D%3D"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	ig     *instagram.Client
	imgDir string
}

func New(api *tgbotapi.BotAPI, serviceURL string) *Bot {
	return &Bot{
		api:    api,
		ig:     instagram.NewClient(serviceURL),
		imgDir: "img",
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
		b.handleURL(msg)
	}
}

func (b *Bot) handleURL(msg *tgbotapi.Message) {
	text := strings.TrimSpace(msg.Text)

	shortcode, ok := instagram.ExtractShortcode(text)
	if !ok {
		b.send(tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, отправьте корректную ссылку на пост или Reel из Instagram."))
		return
	}

	post, err := b.ig.Fetch(shortcode)
	if err != nil {
		b.reportError(msg.Chat.ID, "получить данные поста", err)
		return
	}

	imgData, err := b.ig.DownloadImage(post.ImageURL)
	if err != nil {
		b.reportError(msg.Chat.ID, "скачать изображение", err)
		return
	}

	caption := instagram.CleanCaption(post.Caption)
	source := fmt.Sprintf("---\nИсточник: %s", text)

	if len([]rune(caption)) <= captionLimit {
		b.sendPhotoWithCaption(msg.Chat.ID, imgData, caption+"\n"+source)
	} else {
		runes := []rune(caption)
		firstPart := string(runes[:captionLimit])
		rest := strings.TrimSpace(string(runes[captionLimit:]))

		b.sendPhotoWithCaption(msg.Chat.ID, imgData, firstPart)
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
		// fall back to plain text if images are missing
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
