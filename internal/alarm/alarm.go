package alarm

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// RandomAlarm plays random file from ./sounds directory
func RandomAlarm() {
	files, err := ioutil.ReadDir("sounds")
	if len(files) == 0 {
		log.Fatalln("Sounds catalog is empty")
	}

	f, err := os.Open("sounds/" + files[rand.Intn(len(files))].Name())
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}
