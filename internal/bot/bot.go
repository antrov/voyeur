package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// TelegramBot represents telegram as interaction service
type TelegramBot struct {
	botAPI *tgbotapi.BotAPI
}

// New returns a telegram bot instance
func New(token string) *TelegramBot {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	return &TelegramBot{
		botAPI: bot,
	}
}

// Start bot polling for new commands
func (bot *TelegramBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		var msg tgbotapi.Chattable

		// msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {
		case "help":
			msg = createHelpMsg(update.Message.Chat.ID)
			// msg.Text = "type /sayhi or /status."
		case "mute":
			// msg.Text = "Hi :)"
		case "unmute":

		case "setvolume":

		case "enable":

		case "disable":

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

		if _, err := bot.botAPI.Send(msg); err != nil {
			log.Panic(err)
		}
	}

	log.Println("Stopping TelegramBot")
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

// 		if update.Message.Photo == nil || len(*update.Message.Photo) == 0 {
// 			continue
// 		}

// 		drawPhoto := &(*update.Message.Photo)[0]

// 		for _, photo := range *update.Message.Photo {
// 			if photo.Height > drawPhoto.Height {
// 				drawPhoto = &photo
// 			}
// 		}

// 		url, err := bot.GetFileDirectURL(drawPhoto.FileID)

// 		response, e := http.Get(url)
// 		if e != nil {
// 			log.Fatal(e)
// 		}
// 		defer response.Body.Close()

// 		data, err := ioutil.ReadAll(response.Body)

// 		draw, err := gocv.IMDecode(data, gocv.IMReadUnchanged)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		defer draw.Close()

// 		mask := gocv.NewMat()
// 		defer mask.Close()

// 		preview := gocv.NewMat()
// 		defer preview.Close()

// 		createMask(draw, draw, &mask, &preview)

// 		gocv.IMWrite("test.jpg", preview)
// 		config := tgbotapi.NewPhotoUpload(411400390, "test.jpg")
// 		bot.Send(config)
// 		os.Remove("test.jpg")
// 	}
// }
