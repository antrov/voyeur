package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gitlab.com/antrov/couch-watch/internal/alarm"
	"gitlab.com/antrov/couch-watch/internal/cam"
)

const (
	detectionThreshold    = time.Duration(250 * time.Millisecond)
	cancellationThreshold = time.Duration(600 * time.Millisecond)
	alarmThreshold        = time.Duration(1 * time.Second)
	recordingDuration     = time.Duration(5 * time.Second)
	waitDuration          = time.Duration(1 * time.Minute)
)

func main() {
	rand.Seed(time.Now().Unix())

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	chatID, err := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)
	if err != nil {
		log.Fatalln(err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	events := make(chan cam.CaptureEvent, 5)
	commands := make(chan cam.CaptureCommandType)
	files := make(chan string)

	ticker := time.NewTicker(time.Millisecond * 100)
	ticks := ticker.C

	var firstDetectionTime *time.Time
	var lastDetectionTime *time.Time
	var detectionAlarmed bool
	var isRecording bool
	var isAlarmMuted bool

	resetDetectionState := func() {
		println("Reset state")
		firstDetectionTime = nil
		lastDetectionTime = nil
		detectionAlarmed = false

		if isRecording {
			isRecording = false
			commands <- cam.CaptureCommandTypeCancelRecording
		}
	}

	// go cam.StartSession(0, events, commands, nil)

	for {
		var msg tgbotapi.Chattable

		select {
		case file := <-files:
			println("new file", file)

			switch filepath.Ext(file) {
			case ".png":
				msg = tgbotapi.NewPhotoUpload(chatID, file)

			case ".mp4":
				msg = tgbotapi.NewVideoUpload(chatID, file)
			}

		case event := <-events:
			// println("cam event", event.Type)

			switch event.Type {
			// case cam.EventTypeCaptureStarted:
			// commands <- cam.CaptureCommandTypeStartDetection

			case cam.EventTypePhotoAvailable:
				msg = tgbotapi.NewPhotoUpload(chatID, event.File)

			case cam.EventTypeRecordingAvailable:
				go compressMovie(event.File, files)

			case cam.EventTypeDetection:
				now := time.Now()

				if firstDetectionTime == nil {
					println("First detection")
					firstDetectionTime = &now
				}

				lastDetectionTime = &now
			}

		case <-ticks:
			if firstDetectionTime == nil {
				continue
			}

			switch {
			case time.Since(*firstDetectionTime) >= waitDuration:
				resetDetectionState()

			case time.Since(*firstDetectionTime) >= recordingDuration:
				if isRecording {
					isRecording = false
					commands <- cam.CaptureCommandTypeStopRecording
				}

			case time.Since(*firstDetectionTime) >= alarmThreshold:
				if !detectionAlarmed {
					detectionAlarmed = true

					if !isAlarmMuted {
						go alarm.RandomAlarm()
					}
				}

			case time.Since(*firstDetectionTime) >= cancellationThreshold && time.Since(*lastDetectionTime) >= cancellationThreshold && !detectionAlarmed:
				resetDetectionState()

			case time.Since(*firstDetectionTime) >= detectionThreshold:
				if !isRecording {
					isRecording = true
					commands <- cam.CaptureCommandTypeStartRecording
				}
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
			} else if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "help":
					msg = createHelpMsg(chatID)

				case "mute":
					isAlarmMuted = true

				case "unmute":
					isAlarmMuted = false

				case "setvolume":

				case "raise":
					go alarm.RandomAlarm()

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
			log.Println(err)
		}

		if fileable, ok := msg.(tgbotapi.VideoConfig); ok {
			if file, ok := fileable.File.(string); ok {
				os.Remove(file)
			}
		}
	}
}

func createHelpMsg(chatID int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, `To start, create new region of interest with /createroi and start watching your couch with /enable.

	*Manage Alarms*
	/mute - disable rising of alarm after detection
	/unmute - enable rising of alarm after detection
	/raise - plays alarm sound once

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

func compressMovie(file string, channel chan string) {
	println("Exporting")
	cmd := exec.Command("ffmpeg", "-i", file, "-vcodec", "libx264", "-f", "mp4", "-y", file+".mp4")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		println("cmd error")
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return
	}

	channel <- file + ".mp4"
	os.Remove(file)
}
