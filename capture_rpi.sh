#/bin/bash

cat cmd/playground/capture_to_file.go | ssh pi@raspberrypi.local 'cat > capture_to_file.go; /usr/local/go/bin/go run ./capture_to_file.go -duration 3'
scp pi@raspberrypi.local:~/capture.* .