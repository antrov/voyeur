package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gitlab.com/antrov/couch-watch/internal/cam"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	chatID, err := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	events := make(chan cam.CaptureEvent)
	commands := make(chan cam.CaptureCommandType)
	files := make(chan string)

	go cam.StartSession(0, events, commands)

	for {
		var msg tgbotapi.Chattable

		select {
		case file := <-files:
			println("new file", file)
			msg = tgbotapi.NewPhotoUpload(chatID, file)

		case event := <-events:
			println("cam event", event.Type)

			switch event.Type {
			case cam.EventTypePhotoAvailable:
				msg = tgbotapi.NewPhotoUpload(chatID, event.File)

			case cam.EventTypeCaptureStopped:

			}

		case update := <-updates:
			if update.Message.Chat.ID != chatID {
				continue
			}

			if update.Message == nil { // ignore any non-Message updates
				continue
			}

			if update.Message.Photo != nil && len(*update.Message.Photo) > 0 {
				msg, err = createROI(update.Message.Photo, bot, chatID)
				if err != nil {
					log.Println(err)
				}
				commands <- cam.CaptureCommandTypePreviewROI
				// os.Remove(file)
			} else if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "help":
					msg = createHelpMsg(chatID)

				case "mute":

				case "unmute":

				case "setvolume":

				case "enable":
					commands <- cam.CaptureCommandTypeStartDetection

				case "disable":
					commands <- cam.CaptureCommandTypeStopDetection

				case "capture":
					commands <- cam.CaptureCommandTypeTakePhoto

				case "preview":
					commands <- cam.CaptureCommandTypePreviewROI

				case "createroi":
					msg = tgbotapi.NewMessage(chatID, "Take following photo and draw over it one or more red polygon. Afterthat, upload your drawing.")
					commands <- cam.CaptureCommandTypeTakePhoto
				default:
				}
			} else {
				log.Println("unexpected message type")
			}
		}

		if msg == nil {
			continue
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func createHelpMsg(chatID int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, `To start, create new region of interest with /createroi and start watching your couch with /enable.

	*Manage Alarms*
	/mute - disable rising of alarm after detection
	/unmute - enable rising of alarm after detection
	/setvolume - change volume of alarm

	*Watch Control*
	/enable - start watching of your ROI
	/disable - stop watching of your ROI
	/capture - just take photo
	/preview - take photo with ROI overlay
	/createroi - create Region Of Interest where deetection would be processed`)
	msg.ParseMode = "Markdown"

	return msg
}

func createROI(photos *[]tgbotapi.PhotoSize, bot *tgbotapi.BotAPI, chatID int64) (tgbotapi.MessageConfig, error) {
	drawPhoto := &(*photos)[0]

	for _, photo := range *photos {
		if photo.Height > drawPhoto.Height {
			drawPhoto = &photo
		}
	}

	url, _ := bot.GetFileDirectURL(drawPhoto.FileID)

	response, err := http.Get(url)
	if err != nil {
		return tgbotapi.MessageConfig{}, err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return tgbotapi.MessageConfig{}, err
	}

	err = cam.NewMask(data)
	if err != nil {
		return tgbotapi.MessageConfig{}, err
	}

	msg := tgbotapi.NewMessage(chatID, "Ok, we parsed your drawing and here you go: your brand new the Region of Intereset")

	return msg, nil
}
