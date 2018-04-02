package main

import (
	"github.com/gorilla/mux"
	"github.com/golang/glog"
	"net/http"
	"fmt"
)

var mainDispatcher *PlayerDispatcher

func PayerController(r *mux.Router) error {
	glog.V(1).Infoln("Registering Player Controller")

	mainDispatcher = NewPlayerDispatcher(NewOmxPlayer())
	go mainDispatcher.StartDispatching()

	// explicitly list commands that are accepted
	for _, acceptableCmd := range []string{"play", "pause", "stop", "forward", "backward"} {
		r.PathPrefix("/api/player/" + acceptableCmd).HandlerFunc(commandHandler(mainDispatcher, acceptableCmd))
	}

	return nil
}

// Build and dispatch PlayerCommand
func commandHandler(dispatcher *PlayerDispatcher, commandType string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cmd := NewPlayerCommand(commandType)

		for k, val := range r.URL.Query() {
			if k == "media" && len(val) > 0 {
				// Convert "media" value into File
				path, err := NewPathFromId(val[0])
				if err != nil {
					failureResponse(r, err, w)
					return
				}

				if cmd.File, err = path.ToFile(true); err != nil {
					failureResponse(r, err, w)
					return
				}

			} else {
				// Fill extra args...
				cmd.Args[k] = val
			}
		}

		// and dispatch!
		dispatcher.commands <- cmd
	}
}

// Commands received from user
type PlayerCommand struct {
	// play, stop, pause (toggle play/pause), forward, backward, position
	Operation string

	// File on which executing the command. Optional because if already playing, it's implicit
	File File

	// Optional extra argument
	Args map[string][]string
}

// Create a simple command
// cmd is the name of the command (i.e.: play, stop, ...)
// args first element can the the File on which executing the command ; then must be odd: [key1, value1, key2, value2]
func NewPlayerCommand(cmd string, args ...interface{}) PlayerCommand {
	command := PlayerCommand{Operation: cmd, Args: make(map[string][]string)}

	// Parse extra args...
	if len(args) > 0 {
		if file, ok := args[0].(File); ok {
			command.File = file
			args = args[1:]
		}

		if len(args)%2 != 0 {
			glog.Fatal("Args must be of odd length, plus an optional File type at the beginning. Size is ", len(args), " : ", args)
		}
		for i := range args {
			if i%2 == 0 {
				command.Args[args[i].(string)] = []string{args[i+1].(string)}
			}
		}
	}

	return command
}

type Player interface {
	// Can run file with given extension
	Accept(ext string) bool
	// Execute requested command
	Execute(command PlayerCommand) error
}

type PlayerDispatcher struct {
	stopIt chan bool

	// Registered players
	Players []Player

	// Commands stack
	commands chan PlayerCommand

	// Player currently in use
	currentPlayer Player
}

// Create and start the dispatcher
func NewPlayerDispatcher(players ... Player) *PlayerDispatcher {
	dispatcher := &PlayerDispatcher{
		Players:  players,
		commands: make(chan PlayerCommand, 10),
		stopIt:   make(chan bool, 1),
	}

	return dispatcher
}

// Process asynchronously the command
func (d *PlayerDispatcher) Dispatch(command PlayerCommand) error {
	if d.commands == nil {
		return fmt.Errorf("dispatcher is now closed and do not accept any other command")
	}
	select {
	case d.commands <- command:
		// command is stacked...
		return nil
	default:
		return fmt.Errorf("can't accept command %s: commands chanel already full", command)
	}

}

// Start dispatching in current process.
func (d *PlayerDispatcher) StartDispatching() {
	for {
		select {
		case command := <-d.commands:
			glog.Info("Processing command ", command)

			if command.File != nil {
				previousPlayer := d.currentPlayer

				d.currentPlayer = d.findAppropriatePlayer(command.File)

				if previousPlayer != nil && previousPlayer != d.currentPlayer {
					fmt.Println("Sending STOP to previous player")
					previousPlayer.Execute(NewPlayerCommand("stop"))
				}
				if d.currentPlayer != nil {
					d.currentPlayer.Execute(command)
				}

			} else if d.currentPlayer != nil {
				d.currentPlayer.Execute(command)
			}

		case <-d.stopIt:
			glog.Info("Stop processing commands as requested.")
			// and do not accept any other commands
			close(d.commands)
			d.commands = nil
			return
		}
	}
}

// Stop goroutine that dispatch & process commands
func (d *PlayerDispatcher) StopDispatching() {
	select {
	case d.stopIt <- true:
		// stopping

	default:
		glog.Info("Dispatcher already stopped or stopping.")
	}
}

func (d *PlayerDispatcher) findAppropriatePlayer(file File) Player {
	ext := file.Path().Ext()
	for _, p := range d.Players {
		if p.Accept(ext) {
			return p
		}
	}

	return nil
}

// Assert if media is playable by main dispatcher
func IsPlayable(m *Media) bool {
	return mainDispatcher != nil && mainDispatcher.findAppropriatePlayer(m) != nil
}


