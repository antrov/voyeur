/*
rand.Seed(time.Now().Unix())

files, err := ioutil.ReadDir("./sounds")
				f, err := os.Open("./sounds/" + files[rand.Intn(len(files))].Name())
				if err != nil {
					log.Fatal(err)
				}

				streamer, format, err := mp3.Decode(f)
				if err != nil {
					log.Fatal(err)
				}
				defer streamer.Close()

				speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
				speaker.Play(streamer)

*/