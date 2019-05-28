# Couch Watch
aka Czajnitoring

## Raspi

Wymaga 2GB SWAPu do budowania

GOCV - go get -> make install
git config --global url."git@gitlab.com:".insteadOf "https://gitlab.com/"
dodać klucz publiczny do gitlaba    
git get couch-watch
go get ./...
(ustawić hasło sudo passwd root)
su
echo bcm2835-v4l2 >> /etc/modules

http://jollejolles.com/installing-ffmpeg-with-h264-support-on-raspberry-pi/


?ffmpeg -i USER_VIDEO.h264 -vcodec copy USER_VIDEO.mp4

### Kalibracja rybiego oka

https://medium.com/@kennethjiang/calibrate-fisheye-lens-using-opencv-333b05afa0b0
DIM=(2592, 1944)
K=np.array([[748.2251398191928, 0.0, 1199.1100305796506], [0.0, 750.5481416416964, 1004.5145449548429], [0.0, 0.0, 1.0]])
D=np.array([[0.10297248806242075], [-0.12165385789939036], [0.07232911929651253], [-0.016852757764288767]])