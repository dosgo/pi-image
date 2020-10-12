# pi-image

This is a Raspberry Pi system backup tool. Back up the system according to the actual size. Support Linux / Raspberry/windows (only support full backup) system.



#  source:

https://github.com/BigBubbleGum/RaspberryBackup/blob/master/rpi-backup.sh

# Compile

git clone https://github.com/dosgo/pi-image
go build

# use
linux

sudo ./pi-image

windows
1. Run pi-image in cmd
2. Use "docker run --rm --privileged=true -v `pwd`:/workdir turee/pishrink-docker pishrink <your-image>.img" to reduce the image


