package main

import (
	"testing"
	"io"
	"bufio"
	"github.com/stretchr/testify/assert"
	"time"
	"fmt"
)

func Test_parseInt(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"it should parse int", args{"42"}, 42},
		{"it should parse 0", args{"0"}, 0},
		{"it should parse 0 padding", args{"07"}, 7},
		{"it should return 0 when empty", args{""}, 0},
		{"it should return 0 when not a number", args{"qwerty"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInt(tt.args.val); got != tt.want {
				t.Errorf("parseInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOmxRelativePosition(t *testing.T) {
	type args struct {
		position string
		absolute bool
	}
	tests := []struct {
		name        string
		args        args
		wantSeconds int
	}{
		{"it should extract the time part", args{"seek 01:10:20", true}, 4220},
		{"it should extract the time part when hours and minutes are nil", args{"seek 00:00:21", true}, 21},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOmxTimePosition(tt.args.position, tt.args.absolute); got.seconds != tt.wantSeconds {
				t.Errorf("NewOmxTimePosition() = %v, want %v", got.seconds, tt.wantSeconds)
			}
		})
	}
}

func Test_omxPlaying_readOutput(t *testing.T) {
	t.Run("it should ends go routine after having parsed the time", func(t *testing.T) {
		player := omxPlaying{}

		reader, writer := io.Pipe()

		calledBack := false
		finished := make(chan bool)
		go func() {
			player.readOutput(bufio.NewScanner(reader), func() { calledBack = true })
			finished <- true
		}()

		writer.Write([]byte("seek 01:02:03"))
		writer.Close()

		select {
		case <-finished:
			assert.Equal(t, player.position.seconds, 3723)
			assert.Equal(t, calledBack, true)

		case <-time.After(1 * time.Second):
			assert.Fail(t, "Failure to end readOutput goroutine in time.")
			assert.Equal(t, player.position.seconds, 3723)
		}
	})

	t.Run("it should receive update several times player position", func(t *testing.T) {
		player := omxPlaying{}

		reader, writer := io.Pipe()

		finished := make(chan bool)
		go func() {
			player.readOutput(bufio.NewScanner(reader), func() {})
			finished <- true
		}()

		writer.Write([]byte("seek 01:0"))
		writer.Write([]byte("2:04\n"))

		// wait position to be updated
		timer := time.NewTimer(time.Second)

	found:
		for {
			select {
			case <-timer.C:
				assert.FailNow(t, fmt.Sprint("Position haven't been updated in expected time. Current position: ", player.position))
				break found

			default:
				if player.position.seconds == 3724 {
					break found

				} else {
					time.Sleep(100 * time.Microsecond)
				}
			}
		}

		writer.Write([]byte("foo bar 5'20\"\n"))
		writer.Write([]byte("foo bar 01:02:05\n"))
		writer.Write([]byte("seek 01:02:06\n"))
		writer.Write([]byte("eek 01:02:07\n"))
		writer.Close()

		select {
		case <-finished:
			assert.Equal(t, 3726, player.position.seconds)

		case <-time.After(1 * time.Second):
			assert.Fail(t, "Failure to end readOutput goroutine in time.")
			assert.Equal(t, 3726, player.position.seconds)
		}
	})
}
