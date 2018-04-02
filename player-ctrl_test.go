package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"time"
	"fmt"
)

type MockPlayer struct {
	mock.Mock
}

func (m *MockPlayer) Accept(ext string) bool {
	args := m.Called(ext)
	return args.Bool(0)
}
func (m *MockPlayer) Execute(command PlayerCommand) error {
	args := m.Called(command)
	return args.Error(0)
}

func TestDispatcher_Lookup(t *testing.T) {
	p1 := new(MockPlayer)
	p1.On("Accept", "mp4").Return(true)
	p1.On("Accept", mock.Anything).Return(false)

	p2 := new(MockPlayer)
	p2.On("Accept", "mp3").Return(true)
	p2.On("Accept", mock.Anything).Return(false)

	d := NewPlayerDispatcher(p1, p2)

	t.Run("dispatcher must find appropriate player", func(t *testing.T) {
		assert.Equal(t, d.findAppropriatePlayer(NewMedia(Path{"", "data", "foo", "bar.mp4"})), p1)
		assert.Equal(t, d.findAppropriatePlayer(NewMedia(Path{"", "data", "foo", "bar.mp3"})), p2)
		assert.Nil(t, d.findAppropriatePlayer(NewMedia(Path{"", "data", "foo", "bar.txt"})))
	})
}

func TestDispatcher_stackCommand(t *testing.T) {
	cmds := make(chan PlayerCommand)

	p1 := new(MockPlayer)
	p1.On("Accept", mock.Anything).Return(true)
	p1.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fmt.Println("Mock player: received command ", args.Get(0))
		command := args.Get(0).(PlayerCommand)
		cmds <- command
		fmt.Println("Mock player: after stacking command ", command)
	})

	d := NewPlayerDispatcher(p1)

	t.Run("dispatcher run all commands asynchronously", func(t *testing.T) {
		dispatched := make(chan bool)
		go func() {
			go d.StartDispatching()

			errors := []error{
				d.Dispatch(NewPlayerCommand("play", NewMedia(Path{"", "data", "", "movie.mp4"}))),
				d.Dispatch(NewPlayerCommand("pause")),
				d.Dispatch(NewPlayerCommand("position", "pos", "1:22:47")),
				d.Dispatch(NewPlayerCommand("stop")),
			}

			for i, e := range errors {
				if e != nil {
					t.Errorf("#%d dispatching failed with error: %s", i, e)
				}
			}

			dispatched <- true
		}()

		// Prevent infinite test if dispatcher is not async!
		select {
		case <-dispatched:
			// nothing, just worked
		case <-time.After(100 * time.Millisecond):
			t.Fatal("waited too long to dispatch 4 commands! PlayerDispatcher.Dispatch is expected to be async!")

		}
	})

	t.Run("all commands must come in order", func(t *testing.T) {
		fmt.Println("Read received commands... (players: ", d.Players, ")")

		expected := []string{"play", "pause", "position", "stop"}
		for i, order := range expected {
			fmt.Println("Expecting ", order, " (#", i, ")")
			select {
			case c := <-cmds:
				assert.Equal(t, c.Operation, order, "#", i, " expected was ", order, " but got ", c.Operation)
				if i == 0 {
					assert.Equal(t, c.File.Path().Name, "movie.mp4")
				} else if i == 2 {
					assert.Equal(t, c.Args["pos"], []string{"1:22:47"})
				}

			case <-time.After(10 * time.Millisecond):
				t.Fatal("Not enough commands are waiting: #", i)
			}
		}
	})
}

func TestDispatcher_multiplePlayer(t *testing.T) {
	cmds1 := make(chan PlayerCommand)
	p1 := new(MockPlayer)
	p1.On("Accept", "mp3").Return(true)
	p1.On("Accept", mock.Anything).Return(false)
	p1.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fmt.Println("Mock player 1: received command ", args.Get(0))
		cmds1 <- args.Get(0).(PlayerCommand)
	})

	cmds2 := make(chan PlayerCommand)
	p2 := new(MockPlayer)
	p2.On("Accept", "mp4").Return(true)
	p2.On("Accept", mock.Anything).Return(false)
	p2.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fmt.Println("Mock player 2: received command ", args.Get(0))
		cmds2 <- args.Get(0).(PlayerCommand)
	})

	d := NewPlayerDispatcher(p1, p2)

	t.Run("commands are dispatched to right players", func(t *testing.T) {
		dispatched := make(chan bool)
		go func() {
			go d.StartDispatching()

			errors := []error{
				d.Dispatch(NewPlayerCommand("play", NewMedia(Path{"", "data", "", "movie.mp4"}))),
				d.Dispatch(NewPlayerCommand("play", NewMedia(Path{"", "data", "", "music.mp3"}))),
				d.Dispatch(NewPlayerCommand("pause")),
			}

			for i, e := range errors {
				if e != nil {
					t.Errorf("#%d dispatching failed with error: %s", i, e)
				}
			}

			dispatched <- true
		}()

		assertRoutine := make(chan bool)
		go func() {
			c1 := <-cmds2
			assert.Equal(t, c1.Operation, "play")
			assert.Equal(t, c1.File.Path().Name, "movie.mp4")

			c2 := <-cmds2
			assert.Equal(t, c2.Operation, "stop")

			c3 := <-cmds1
			assert.Equal(t, c3.Operation, "play")
			assert.Equal(t, c3.File.Path().Name, "music.mp3")

			c4 := <-cmds1
			assert.Equal(t, c4.Operation, "pause")

			assertRoutine <- true
		}()

		select {
		case <-assertRoutine:
			// worked

		case <-time.After(10 * time.Millisecond):
			t.Fatal("All expected messages haven't been received. Check logs to see which have been received.")
		}
	})
}

func TestDispatcher_stopping(t *testing.T) {
	cmds := make(chan PlayerCommand)

	p1 := new(MockPlayer)
	p1.On("Accept", mock.Anything).Return(true)
	p1.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fmt.Println("Mock player: received command ", args.Get(0))
		cmds <- args.Get(0).(PlayerCommand)
	})

	d := NewPlayerDispatcher(p1)

	t.Run("dispatcher do not accept new commands once stopped", func(t *testing.T) {
		endDispatching := make(chan bool)
		dispatched := make(chan bool)
		go func() {
			go func() {
				d.StartDispatching()
				endDispatching <- true
			}()

			d.Dispatch(NewPlayerCommand("play", NewMedia(Path{"", "data", "", "movie.mp4"})))

			c := <-cmds
			assert.Equal(t, c.Operation, "play")

			d.StopDispatching()
			time.Sleep(1 * time.Millisecond)
			err := d.Dispatch(NewPlayerCommand("play", NewMedia(Path{"", "data", "", "movie.mp4"})))

			if err == nil {
				t.Fatal("Was expecting an error while triing to dispatch another message while dispatcher is stopped.")
			}

			select {
			case <-endDispatching:
				// ok
			default:
				t.Fatal("Dispatcher Goroutine should have been terminated")
			}

			dispatched <- true
		}()

		select {
		case <-dispatched:
			return
		case <-time.After(10 * time.Millisecond):
			t.Fatal("TIMEOUT - something went wrong on the path and either messages hasn't been consumed, or something is stuck.")
		}
	})
}