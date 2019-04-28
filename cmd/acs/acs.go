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

	capture, _ := cam.New(0)
	defer capture.Close()

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
	var camEvents chan cam.CaptureEvent
	var processEvents chan string

	for {
		var msg tgbotapi.Chattable

		select {
		case process := <-processEvents:
			println("process events", process)
			msg = tgbotapi.NewPhotoUpload(chatID, process)

		case event := <-camEvents:
			println("cam event", event.Type)

			switch event.Type {
			case cam.EventTypePhotoAvailable:
				msg = tgbotapi.NewPhotoUpload(chatID, event.File)

			case cam.EventTypeCaptureStopped:
				capture.Close()
			}

		case update := <-updates:
			if update.Message.Chat.ID != chatID {
				continue
			}

			if update.Message == nil { // ignore any non-Message updates
				continue
			}

			if update.Message.Photo != nil && len(*update.Message.Photo) > 0 {
				processEvents = make(chan string)
				go createROI(update.Message.Photo, bot, processEvents)
				// os.Remove(file)
			} else if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "help":
					msg = createHelpMsg(chatID)

				case "mute":

				case "unmute":

				case "setvolume":

				case "enable":
					camEvents = capture.StartDetection()

				case "disable":
					capture.Close()

				case "capture":
					capture.TakePhoto()

				case "createroi":
					capture.TakePhoto()
					// msg.Text = "I'm ok."
				default:
					// msg.Text = "I don't know that command"
				}
			} else {
				log.Println("unexpected message type")
			}
		}

		if msg == nil {
			// log.Println("msg empty")
			continue
		}

		// go func(msg tgbotapi.Chattable) {
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
		// }(msg)

	}

	// var botUpdates chan bot

}

func createHelpMsg(chatID int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, `To start, create new region of interest with /createroi and start watching your couch with /enable.

	*Manage Alarms*
	/mute - disable rising of alarm after detection
	/unmute - enable rising of alarm after detection
	/setvolume - change volume of alarm

	*Watch Control*
	/enable - start watching of your roi
	/disable - stop watching of your roi
	/capture - just take photo
	/createroi - create Region Of Interest where deetection would be processed`)
	msg.ParseMode = "Markdown"

	return msg
}

func createROI(photos *[]tgbotapi.PhotoSize, bot *tgbotapi.BotAPI, channel chan string) {
	drawPhoto := &(*photos)[0]

	for _, photo := range *photos {
		if photo.Height > drawPhoto.Height {
			drawPhoto = &photo
		}
	}

	url, _ := bot.GetFileDirectURL(drawPhoto.FileID)

	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	file, err := cam.NewMask(data)
	if err != nil {
		log.Fatal(err)
	}

	channel <- file
}
