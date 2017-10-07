package main

import (
	"github.com/gorilla/mux"
	"github.com/golang/glog"
	"net/http"
	"os/exec"
)

func PayerController(r *mux.Router) error {

	glog.V(1).Infoln("Registering Player Controller")
	r.PathPrefix("/player/play").HandlerFunc(PlayMedia)

	return nil
}

func PlayMedia(w http.ResponseWriter, r *http.Request) {
	mediaPublicPath := r.URL.Query()["media"][0]
	glog.Infoln("Playing media: ", mediaPublicPath)

	requested := PathRequest{Host:r.Host}
	requested.ParsePath(mediaPublicPath)

	media := newMediaFromRequest(&requested)
	glog.Infoln("File is: ", media.localPath)

	media.Play()
}
func (m *Media) Play() error{
	// TODO mage error, test if file exist...
	//return exec.Command("mplayer", m.localPath).Run()
	return exec.Command("omxplayer", "-o", "hdmi", m.localPath).Run()
}
