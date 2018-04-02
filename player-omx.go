package main

import (
	"os/exec"
	"strings"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

type OmxPlayer struct {
	process *exec.Cmd
	playing File
}

func NewOmxPlayer() *OmxPlayer {
	return &OmxPlayer{}
}

// Film files are playable
func (player *OmxPlayer) Accept(ext string) bool {
	lext := strings.ToLower(ext)
	return lext == "mkv" || lext == "mp4" || lext == "avi"
}

func (player *OmxPlayer) Execute(command PlayerCommand) error {
	playCmd := command.Operation == "play" && command.File != nil && (player.playing == nil || command.File.Path() != player.playing.Path())

	if command.Operation == "stop" || playCmd {
		if player.process != nil {
			in, err := player.process.StdinPipe()
			if err != nil {
				return err
			}
			in.Write([]byte{'s'})
		}
	}

	if playCmd {
		glog.Info("Start to play ", command.File.Path().localPath)
		player.playing = command.File
		player.process = exec.Command("omxplayer", "-player", "hdmi", command.File.Path().localPath)

		if err := player.process.Start(); err != nil {
			return err
		}

		// TODO Listen for omx updates

	}

	return errors.New(fmt.Sprintf("Command %s is not implemented by OmxPlayer adapter.", command))
}
