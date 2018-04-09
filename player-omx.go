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
	return lext == "mkv" || lext == "mp4" || lext == "avi" || lext == "mov"
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
			player.instance.TogglePause()

		case ope == "forward":
			player.instance.omxExec('\033', '[', 'C')

		case ope == "backward":
			player.instance.omxExec('\033', '[', 'D')

		case ope == "bigForward":
			player.instance.omxExec('\033', '[', 'A')

		case ope == "bigBackward":
			player.instance.omxExec('\033', '[', 'B')

		default:
			return errors.New(fmt.Sprintf("Command %s is not implemented by OmxPlayer adapter.", command))
		}
	}

	if playCmd {
		file := command.File.Path().localPath
		glog.Info("Start to play ", file)

		process := exec.Command("stdbuf", "-oL", "-eL", "omxplayer", "-o", "hdmi", file)
		reader, _ := process.StdoutPipe()
		process.Stderr = process.Stdout

		player.instance = &omxPlaying{
			playing:  command.File,
			process:  process,
			position: NewTimePosition(0, 0, 0, false),
			Length:   NewTimePosition(0, 0, 0, true),
		}

		// Start listening for updates (position in media)
		go player.instance.readOutput(bufio.NewScanner(reader), func() {
			glog.Info("Finished to play ", file)
			player.instance = nil
		})
		go player.instance.readMediaLength(file)

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

	return NewPlayerStatus(player.instance.playing, player.instance.Paused, player.instance.position, player.instance.Length)
}

// Playing instance of OMX Player
type omxPlaying struct {
	process  *exec.Cmd
	playing  File
	stdin    io.WriteCloser
	position TimePosition

	Paused bool
	Length TimePosition
}

// Pass a command (key) to OMX Player
func (player *omxPlaying) omxExec(key ...byte) {
	player.stdin.Write(key)
}

// Read OMX Player output
func (player *omxPlaying) readOutput(scanner *bufio.Scanner, callback func()) {
	for scanner.Scan() {
		line := scanner.Text()
		glog.V(2).Info("[omxplayer] stdout: ", line)

		if strings.HasPrefix(line, "Seek") {
			player.position = NewOmxTimePosition(line, false)
		}
	}

	callback()
}

// Use ffmepg to get media length
func (player *omxPlaying) readMediaLength(file string) {

	process := exec.Command("ffmpeg", "-i", file)
	reader, _ := process.StdoutPipe()
	process.Stderr = process.Stdout

	if err := process.Start(); err != nil {
		glog.Error("Can't determine media length with ffmpeg (start): ", err)
	}

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		glog.V(2).Info("[ffmpeg] stdout: ", line)

		if strings.Index(line, "Duration") >= 0 {
			player.Length = NewOmxTimePosition(line, true)
			glog.Info("Media ", file, " length is ", player.Length)
		}
	}

	glog.V(2).Info("ffmpeg goroutine ends.")
}

// Toggle pause and fix position to not keep it running
func (player *omxPlaying) TogglePause() {
	player.Paused = !player.Paused
	player.position = player.position.Absolute(player.Paused)
}

// Parse "seek" line from OMX Player output
func NewOmxTimePosition(line string, absolute bool) TimePosition {
	pattern := regexp.MustCompile(`(\d+):(\d+):(\d+)`)

	times := pattern.FindStringSubmatch(line)
	if times == nil {
		return NewTimePosition(0, 0, 0, absolute)
	}

	return NewTimePosition(parseInt(times[1]), parseInt(times[2]), parseInt(times[3]), absolute)
}

// Parse int from string, ignore error and return 0
func parseInt(val string) int {
	if n, e := strconv.Atoi(val); e == nil {
		return n
	}

	return 0
}