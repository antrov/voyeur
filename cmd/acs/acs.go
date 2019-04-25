package main

import (
	"log"
	"os"

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

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)
	var camEvents chan cam.CaptureEvent

	for {
		select {
		case event := <-camEvents:
			println("cam event", event.Message)

		case update := <-updates:
			if update.Message == nil { // ignore any non-Message updates
				continue
			}

			var msg tgbotapi.Chattable

			// msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			switch update.Message.Command() {
			case "help":
				msg = createHelpMsg(update.Message.Chat.ID)

			case "mute":

			case "unmute":

			case "setvolume":

			case "enable":
				camEvents = capture.StartDetection()

			case "disable":
				capture.Close()

			case "capture":

			case "createroi":
				// msg.Text = "I'm ok."
			default:
				// msg.Text = "I don't know that command"
			}

			if msg == nil {
				log.Println("msg empty")
				continue
			}

			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
		}
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
