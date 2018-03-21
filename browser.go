package main

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"

	"strings"
	"encoding/json"

)

const BROWSER_PREFIX = "/api/browser"

func BrowserController(r *mux.Router) error {
	glog.V(1).Infoln("Registering Browser Controller")

	err := ConfigureRoots()
	if err != nil {
		return err
	}

	r.PathPrefix(BROWSER_PREFIX).HandlerFunc(ShowMedia)

	glog.Infoln("Browser Controller config loaded. Roots are:", roots)
	return nil
}

// Handle browsing request
func ShowMedia(w http.ResponseWriter, r *http.Request) {
	path, err1 := parsePath(r)
	if err1 != nil {
		failureResponse(r, err1, w)
	}

	file, err2 := path.ToFile(false)
	if err2 != nil {
		failureResponse(r, err1, w)
	}

	if err1 == nil && err2 == nil {
		respondWithJSON(w, 200, NewFileDto(file))
	}
}

type FileDto struct {
	Type     string `json:"type"`
	PathId   string `json:"pathId"`
	ParentId string `json:"parentId"`
	Name     string `json:"name"`

	// Specific for directories
	Children []FileDto `json:"children,omitempty"`

	// Specific to Media
	Playable bool `json:"playable"`
}

func NewFileDto(file File) FileDto {
	dto := FileDto{
		Type:     file.Type(),
		PathId:   file.Path().PathId(),
		ParentId: file.Path().ParentId(),
		Name:     file.Path().DisplayName(),
	}

	if dir, ok := file.(*Dir); ok {
		dto.Children = make([]FileDto, len(dir.Children))
		for i, c := range dir.Children {
			dto.Children[i] = NewFileDto(c)
		}
	}

	if media, ok := file.(*Media); ok {
		dto.Playable = media.IsPlayable()
	}

	return dto
}

func failureResponse(r *http.Request, err error, w http.ResponseWriter) {
	glog.Warning("Fail to browse '", r.URL.Path, "': ", err.Error())
	respondWithJSON(w, 500, map[string]string{"error": err.Error()})
}

// Parse file public path and resolve its internal path
func parsePath(request *http.Request) (Path, error) {
	publicPath := strings.Trim(strings.TrimPrefix(request.URL.Path, BROWSER_PREFIX), "/")

	var root string
	var relativePath string
	var name string

	if strings.Contains(publicPath, "/") {
		firstSlash := strings.Index(publicPath, "/")
		lastSlash := strings.LastIndex(publicPath, "/")

		root = publicPath[:firstSlash]
		if firstSlash < lastSlash {
			relativePath = publicPath[firstSlash+1:lastSlash]
		}
		name = publicPath[lastSlash+1:]

	} else if publicPath != "" {
		// Simple root
		root = publicPath
	}
	// else it's browser index

	return NewPath(root, relativePath, name)
}

// Serialise payload into JSON format, respond with 500 if can't serialise to JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	glog.V(1).Infoln("Marshaled: ", string(response), " (payload=", payload, ")")

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write(response)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		mess := `{"error", "Can not serialise in JSON document provided: ` + err.Error() + `"}`
		w.Write([]byte(mess))
	}
}
