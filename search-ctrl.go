package main

import (
	"github.com/gorilla/mux"
	"github.com/golang/glog"
	"net/http"
	"os"
	"sync"
	"path/filepath"
	"time"
	"strings"
	"sort"
)

// Add to API a search endpoint
func SearchController(r *mux.Router) error {
	glog.V(1).Infoln("Registering Search Controller")

	r.PathPrefix("/api/search").Queries("pattern", "").HandlerFunc(SearchMedia)

	return nil
}

func SearchMedia(writer http.ResponseWriter, request *http.Request) {
	patterns := request.URL.Query()["pattern"]
	if len(patterns) <= 0 || len(patterns[0]) < 3 {
		respondWithJSON(writer, 400, map[string]string{"error": "'pattern' query parameter is required and must at least have 3 chars"})
		return
	}

	filter := func(name string) bool {
		return filterName(patterns[0], name)
	}

	files := StartSearching(filter, getRoots())
	glog.Info("Search of ", patterns[0], " returned ", len(files), " medias.")
	respondWithJSON(writer, 200, files)
}

// Get roots using public model functions
func getRoots() map[string]string {
	r := make(map[string]string, len(roots))
	for name, p := range roots {
		r[name] = p.localPath
	}

	return r
}

// Test if pattern is found in given name
func filterName(pattern string, name string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(pattern))
}

const (
	searchLoaderRoutine = 2
	searchBuffer        = 50
	searchBatch         = 10
)

type NamePredicate func(name string) bool

type fileSearch struct {
	searchLoaderRoutine int
	searchBuffer        int
	searchBatch         int

	foundFileIds chan string
	foundMedia   chan File

	// Media loader function
	mediaLoader func(buffer [64]string, length int, files chan File)
}

func StartSearching(acceptanceCriteria NamePredicate, roots map[string]string) []FileDto {

	fileSearch := fileSearch{
		foundFileIds: make(chan string, searchBuffer),
		foundMedia:   make(chan File, searchBuffer),
		mediaLoader:  loadBatch,

		searchLoaderRoutine: searchLoaderRoutine,
		searchBuffer:        searchBuffer,
		searchBatch:         searchBatch,
	}

	// Start response builder routine - stop when chanel is closed (down this method)
	response := make(chan []FileDto)
	go fileSearch.buildResponse(response)

	// Start loaders routines - stop driven by global quit
	wgLoaders := new(sync.WaitGroup)
	wgLoaders.Add(searchLoaderRoutine)
	for i := 0; i < searchLoaderRoutine; i++ {
		go fileSearch.startLoading(wgLoaders)
	}

	// Start file walkers - stop by themselves
	wgWalkers := new(sync.WaitGroup)
	wgWalkers.Add(len(roots))
	for root, path := range roots {
		go fileSearch.walkThrow(root, path, acceptanceCriteria, wgWalkers)
	}

	// Waiting end of routines
	wgWalkers.Wait()
	close(fileSearch.foundFileIds)
	wgLoaders.Wait()
	close(fileSearch.foundMedia)

	// Return completed results
	return <-response
}

// scan recursively all files, add in channel non-dir file accepted by criteria
func (s *fileSearch) walkThrow(root string, rootPath string, acceptanceCriteria NamePredicate, group *sync.WaitGroup) {
	defer group.Done()

	err := filepath.Walk(rootPath, func(path string, f os.FileInfo, err error) error {
		switch {
		case f == nil:
			glog.Warning("Can't stats file '"+path+"': ", err)
			return nil

		case f.IsDir() && strings.HasPrefix(path, "."):
			// Skip hidden files
			return filepath.SkipDir

		case !f.IsDir() && acceptanceCriteria(f.Name()):
			s.foundFileIds <- root + "/" + strings.Trim(strings.TrimPrefix(path, rootPath), "/")
			return nil

		default:
			return nil
		}
	})

	if err != nil {
		glog.Error("Couldn't complete ", rootPath, " scan because: ", err, ".")
	}
}

// Group results by batch and delegate conversion into Media
func (s *fileSearch) startLoading(group *sync.WaitGroup) {
	defer group.Done()

	var buffer [64]string

	for {
		length := 0

		f, ok := <-s.foundFileIds
		if ok {
			buffer[length] = f
			length++
		} else {
			return
		}

		// then, wait to get buffer full or timeout
		timer := time.NewTimer(100 * time.Millisecond)
		closed := false
	aggregating:
		for !closed && length < s.searchBatch {
			select {
			case f, ok := <-s.foundFileIds:
				if ok {
					buffer[length] = f
					length++
				} else {
					closed = true
				}

			case <-timer.C:
				break aggregating
			}
		}

		// Load batch
		s.mediaLoader(buffer, length, s.foundMedia)
	}
}
func (s *fileSearch) buildResponse(response chan []FileDto) {
	// TRUE if f1 > f2 FIXME this code already somewhere else
	compareFile := func(f1, f2 FileDto) bool { return strings.ToLower(f1.Name) > strings.ToLower(f2.Name) }

	// Note: would certainly be faster to sort at the end using sorting algorithm!
	var medias []FileDto
	for media := range s.foundMedia {
		dto := NewFileDto(media)

		index := sort.Search(len(medias), func(i int) bool { return compareFile(medias[i], dto) })
		medias = append(medias, FileDto{})
		copy(medias[index+1:], medias[index:])
		medias[index] = dto
	}

	response <- medias
	glog.Info("Ends of buildResponse")
}

func loadBatch(buffer [64]string, length int, files chan File) {
	for i := 0; i < length; i++ {
		pathId := buffer[i]

		// TODO (remove root, select name, keep rest as middle path)
		if path, err := NewPathFromId(pathId); err == nil {
			if media, err := path.ToFile(true); err == nil {
				files <- media
			} else {
				glog.Warning("Can't create Media for path ", path, " : ", err)
			}
		} else {
			glog.Warning("Can't create File for pathId ", pathId, " : ", err)
		}
	}
}
