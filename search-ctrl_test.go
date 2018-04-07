package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_fileSearch_walkThrow(t *testing.T) {
	t.Run("it should find 2 files starting search-ctrl in current dir", func(t *testing.T) {
		s := fileSearch{
			foundFileIds: make(chan string, 1024),
		}
		wg := new(sync.WaitGroup)
		wg.Add(1)
		finished := make(chan bool)
		go func() {
			wg.Wait()
			finished <- true
		}()

		// When
		predicate := func(name string) bool {
			return strings.HasPrefix(name, "search-ctrl")
		}
		s.walkThrow("foobar", workingDir(), predicate, wg)

		// Then
		close(s.foundFileIds)
		expected := []string{"foobar/search-ctrl.go", "foobar/search-ctrl_test.go"}
		i := 0
		for f := range s.foundFileIds {
			assert.Equal(t, expected[i], f, "File #%d is expected to be %s, was %s", i, expected[i], f)
			i++
		}

		select {
		case f := <-finished:
			assert.Equal(t, true, f)
		case <-time.After(time.Millisecond):
			assert.Fail(t, "Wait Group should have been closed.")
		}
	})
}

func Test_fileSearch_startLoading(t *testing.T) {

	t.Run("it should create 2 batches with 3 elem max in it", func(t *testing.T) {
		s, batches := newFileSearch()

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go s.startLoading(wg)

		s.foundFileIds <- "foobar/ironman"
		s.foundFileIds <- "foobar/thor"
		s.foundFileIds <- "foobar/hawkeye"
		s.foundFileIds <- "foobar/hulk"

		close(s.foundFileIds)
		wg.Wait()

		expected := [][]string{
			{"foobar/ironman", "foobar/thor", "foobar/hawkeye"},
			{"foobar/hulk"},
		}
		assert.Equal(t, expected, batches.batches)
	})

	t.Run("it should timeout after 100ms and launch next batch anyway", func(t *testing.T) {
		s, batches := newFileSearch()

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go s.startLoading(wg)

		// not full batch
		s.foundFileIds <- "foobar/ironman"
		s.foundFileIds <- "foobar/thor"

		// Wait for timeout
		time.Sleep(150 * time.Millisecond)

		expected := [][]string{
			{"foobar/ironman", "foobar/thor"},
		}
		assert.Equal(t, expected, batches.batches)

		close(s.foundFileIds)
	})

	t.Run("it should ends goroutine when channel is closed", func(t *testing.T) {
		s, _ := newFileSearch()

		finished := make(chan bool)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			s.startLoading(wg)
			finished <- true
		}()

		// ends goroutine
		close(s.foundFileIds)

		select {
		case <-finished:
			// ok
		case <-time.After(10 * time.Millisecond):
			assert.Fail(t, "Closing channel should have terminate the go routine as well (even on first element)")
		}

	})
}

type Batches struct {
	batches [][]string
}

func newFileSearch() (fileSearch, *Batches) {
	batches := new(Batches)
	s := fileSearch{
		foundFileIds: make(chan string, 10),
		foundMedia:   make(chan File, 10),
		searchBatch:  3,
		mediaLoader: func(buffer [64]string, length int, files chan File) {
			fmt.Println("MOCK - Load buffer (", length, "): ", buffer)
			array := make([]string, length)
			for i := 0; i < length; i++ {
				array[i] = buffer[i]
			}
			batches.batches = append(batches.batches, array)
		},
	}

	return s, batches
}

func Test_fileSearch_buildResponse(t *testing.T) {
	s := fileSearch{
		foundMedia: make(chan File, 10),
	}

	resp := make(chan []FileDto)
	go s.buildResponse(resp)

	s.foundMedia <- NewMedia(Path{Name: "foo.avi"})
	s.foundMedia <- NewMedia(Path{Name: "Gif.avi"})
	s.foundMedia <- NewMedia(Path{Name: "Echo_2018.mp4"})
	s.foundMedia <- NewMedia(Path{Name: ".victoria-secret.mp4"})

	close(s.foundMedia)

	select {
	case medias := <-resp:
		assert.Equal(t, 4, len(medias))
		for i, v := range []string{".victoria-secret.mp4", "Echo_2018.mp4", "foo.avi", "Gif.avi"} {
			assert.Equal(t, v, medias[i].Name, "#%d expected to be %s but is %s", i, v, medias[i].Name)
		}

	case <-time.After(10 * time.Millisecond):
		assert.Fail(t, "Never received list of sorted media...")
	}
}

func Test_filterName(t *testing.T) {
	type args struct {
		pattern string
		name    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"it should match when equals", args{"foo", "foo"}, true},
		{"it should match when starts", args{"foo", "foobar"}, true},
		{"it should match when ends", args{"bar", "foobar"}, true},
		{"it should match when middle", args{"bar", "foobarbaz"}, true},
		{"it should be case insensitive", args{"BAr", "fooBaRbaz"}, true},
		{"it should not match when not contained", args{"fobar", "foobarbaz"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterName(tt.args.pattern, tt.args.name); got != tt.want {
				t.Errorf("filterName() = %v, want %v", got, tt.want)
			}
		})
	}
}
