package main

import (
	"os/exec"
	"strings"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io"
	"bufio"
	"regexp"
	"strconv"
)

type OmxPlayer struct {
	instance *omxPlaying
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
	// New play, or play of another media
	playCmd := command.Operation == "play" && command.File != nil && (player.instance == nil || command.File.Path() != player.instance.playing.Path())

	if player.instance != nil {
		// Commands on current play instance

		ope := command.Operation
		switch {
		case ope == "stop" || playCmd:
			glog.Info("Stopping ", player.instance.playing.Path().localPath)
			player.instance.omxExec('q')

		case ope == "pause":
			player.instance.omxExec('p')

		case ope == "forward":
			player.instance.omxExec('\033', '[', 'C')

		case ope == "backward":
			player.instance.omxExec('\033', '[', 'D')

		case ope == "bigForward":
			player.instance.omxExec('\033', '[', 'A')

		case ope == "bigBackward":
			player.instance.omxExec('\033', '[', 'B')

		default:
			if !playCmd {
				return errors.New(fmt.Sprintf("Command %s is not implemented by OmxPlayer adapter.", command))
			}
		}
	}

	if playCmd {
		glog.Info("Start to play ", command.File.Path().localPath)

		process := exec.Command("stdbuf", "-oL", "-eL", "omxplayer", "-o", "hdmi", command.File.Path().localPath)
		reader, _ := process.StdoutPipe()
		process.Stderr = process.Stdout

		initialPosition := NewRelativePosition(0, 0, 0)
		player.instance = &omxPlaying{
			playing:  command.File,
			process:  process,
			position: &initialPosition,
		}

		// Start listening for updates (position in media)
		go player.instance.readOutput(bufio.NewScanner(reader))

		var err error
		if player.instance.stdin, err = process.StdinPipe(); err != nil {
			return err
		}

		if err := process.Start(); err != nil {
			return err
		}
	}

	return nil
}

// Return status of OMX Player
func (player *OmxPlayer) GetStatus() PlayerStatus {
	if player.instance == nil || player.instance.position == nil {
		return NotPlayingStatus()
	}

	return NewPlayerStatus(player.instance.playing, *player.instance.position)
}

// Playing instance of OMX Player
type omxPlaying struct {
	process  *exec.Cmd
	playing  File
	stdin    io.WriteCloser
	position *RelativePosition
}

// Pass a command (key) to OMX Player
func (player *omxPlaying) omxExec(key ...byte) {
	player.stdin.Write(key)
}

// Read OMX Player output
func (player *omxPlaying) readOutput(scanner *bufio.Scanner) {
	for scanner.Scan() {
		line := scanner.Text()

		glog.V(2).Info("OMX Output - ", line)
		if strings.HasPrefix(line, "Seek") {
			player.position = NewOmxRelativePosition(line)
		}
	}
}

// Parse "seek" line from OMX Player output
func NewOmxRelativePosition(position string) *RelativePosition {
	pattern := regexp.MustCompile(`(\d+):(\d+):(\d+)`)

	times := pattern.FindStringSubmatch(position)
	if times == nil {
		return nil
	}

	pos := NewRelativePosition(parseInt(times[1]), parseInt(times[2]), parseInt(times[3]))
	return &pos
}

// Parse int from string, ignore error and return 0
func parseInt(val string) int {
	if n, e := strconv.Atoi(val); e == nil {
		return n
	}

	return 0
}