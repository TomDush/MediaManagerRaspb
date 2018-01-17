# How to stream with VLC

## Server:

Start VLC:

    vlc --ttl 12 -vvv --color -I telnet --telnet-password pass --rtsp-host 0.0.0.0 --rtsp-port 5554


Then: 

dush@dush-server ~ $ telnet localhost 4212
Trying ::1...
Connected to localhost.
Escape character is '^]'.
VLC media player 2.2.8 Weatherwax
Password: 
Wrong password
Password: 
Welcome, Master
> new StarTrek vod enabled
new
> setup StarTrek input /mnt/data/Media/Movies/Sagas/Star\ Trek/Star.Trek.2009.1080p.MULTi.BluRay.x264-ForceBleue.mkv
setup
> ^C


## Client:

VLC:

    vlc --ttl 12 -vvv --color -I telnet --telnet-password pass --rtsp-host 0.0.0.0 --rtsp-port 5554

OMX:

    omxplayer -o hdmi rtsp://dush-server:5554/StarTrek

