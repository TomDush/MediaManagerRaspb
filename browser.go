package main

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"

	"strings"
	"fmt"
	"encoding/json"

	"os"
	"io/ioutil"
	"net/url"
)

var roots map[string]string = make(map[string]string)

const BROWSER_PREFIX = "/api/browser"

func BrowserController(r *mux.Router) error {
	glog.V(1).Infoln("Registering Browser Controller")

	err := configure()
	if err != nil {
		return err
	}

	r.PathPrefix(BROWSER_PREFIX).HandlerFunc(ShowMedia)

	glog.Infoln("Browser Controller config loaded. Roots are:", roots)
	return nil
}

// Load roots configuration
func configure() error {
	config := GetMmConfig()
	rootConfig := strings.Split(config.roots, ",")
	if len(rootConfig) == 0 {
		return fmt.Errorf("Media roots must be required (-roots key1:/path,key2:/path/2) ")
	}
	for i := 0; i < len(rootConfig); i++ {
		r := strings.Split(rootConfig[i], ":")
		glog.V(1).Infoln("Parsing ", r)
		if len(r) != 2 {
			return fmt.Errorf("roots configuration invalid '%s', it must be name=path", r)
		}
		roots[r[0]] = r[1]
	}

	return nil
}

// Handle browsing request
func ShowMedia(w http.ResponseWriter, r *http.Request) {
	pathRequest := NewPathRequest(r)

	var elem interface{}
	var err error

	if pathRequest.IsIndex() {
		publicUrl := pathRequest.PublicUrl()
		index := Dir{Name: "Home", DetailUrl: publicUrl}
		for root, _ := range roots {
			index.Children = append(index.Children, Dir{Name: root, Root: root, DetailUrl: publicUrl + root, PathId: root})
		}

		elem = index
	} else {
		elem, err = BrowseTo(&pathRequest)
	}

	if err == nil {
		err = respondWithJSON(w, 200, elem)
	}

	if err != nil {
		glog.Warning("Fail to browse '", r.URL.Path, "': ", err.Error())
		respondWithJSON(w, 500, map[string]string{"error": err.Error()})
	}
}

// Instantiate a Media or a Dir depending on the targetted file.
func BrowseTo(pathRequest *PathRequest) (interface{}, error) {
	var elem interface{}

	glog.V(2).Infoln("Stats of ", pathRequest.LocalPath())
	stat, err := os.Stat(pathRequest.LocalPath())
	if err == nil {
		if stat.IsDir() {
			glog.V(1).Infoln("Browse directory ", pathRequest.LocalPath())
			elem, err = NewDir(pathRequest)

		} else {
			glog.V(1).Infoln("Get media details: ", pathRequest.LocalPath())
			elem = newMediaFromRequest(pathRequest)
		}
	}

	return elem, err
}

// Parsed URL with convenient method to regenerate URL, partial path, ...
type PathRequest struct {
	Host string

	Root         string
	Name         string
	RelativePath string

	// Absolute local path
	localPath string
}

func NewPathRequest(request *http.Request) PathRequest {
	r := PathRequest{}

	r.Host = request.Host

	r.ParsePath(strings.Trim(strings.TrimPrefix(request.URL.Path, BROWSER_PREFIX), "/"))

	return r
}

// Parse file public path and resolve its internal path
func (r *PathRequest) ParsePath(publicPath string) {
	if strings.Contains(publicPath, "/") {
		firstSlash := strings.Index(publicPath, "/")
		lastSlash := strings.LastIndex(publicPath, "/")

		r.Root = publicPath[:firstSlash]
		if firstSlash < lastSlash {
			r.RelativePath = publicPath[firstSlash+1:lastSlash]
		}
		r.Name = publicPath[lastSlash+1:]

	} else if publicPath != "" {
		// Simple root
		r.Root = publicPath
	}
	// else it's browser index
}
func (r *PathRequest) LocalPath() string {
	if r.localPath == "" {
		r.localPath = joinNotEmpty([]string{roots[r.Root], r.RelativePath, r.Name}, "/")
	}

	return r.localPath
}
func (r *PathRequest) IsIndex() bool {
	return r.Root == ""
}
func (r *PathRequest) PublicPath() string {
	return joinNotEmpty([]string{r.Root, r.RelativePath, r.Name}, "/")
}
func (r *PathRequest) PublicUrl() string {
	publicUrl := url.URL{Scheme: "http", Host: r.Host, Path: BROWSER_PREFIX + "/" + r.PublicPath()}
	return publicUrl.String()
}
func (r *PathRequest) ParentPublicUrl() string {
	publicUrl := url.URL{Scheme: "http", Host: r.Host, Path: joinNotEmpty([]string{BROWSER_PREFIX, r.Root, r.RelativePath}, "/")}
	if r.IsRoot() {
		publicUrl = url.URL{Scheme: "http", Host: r.Host, Path: BROWSER_PREFIX}
	}
	return publicUrl.String()
}
func (r *PathRequest) IsRoot() bool {
	return r.Root != "" && r.Name == ""
}

type Dir struct {
	Name      string `json:"name"`
	Root      string `json:"root,omitempty"`
	PathId    string `json:"pathId"`
	localPath string `json:"localPath,omitempty"`
	ParentId  string `json:"parentId,omitempty"`

	ParentUrl string `json:"parentUrl,omitempty"`
	DetailUrl string `json:"detailUrl"`

	Children []interface{} `json:"children,omitempty"`
}

func NewDir(r *PathRequest) (dir Dir, err error) {
	parentId := joinNotEmpty([]string{r.Root, r.RelativePath}, "/")
	if r.IsRoot() {
		parentId = ""
	}

	dir = Dir{Root: r.Root, localPath: r.localPath, Name: r.Name, PathId: joinNotEmpty([]string{r.Root, r.RelativePath, r.Name}, "/"), ParentId: parentId}
	dir.DetailUrl = r.PublicUrl()
	dir.ParentUrl = r.ParentPublicUrl()

	if r.IsRoot() {
		dir.Name = r.Root
	}

	files, err := ioutil.ReadDir(r.localPath)

	if err == nil {
		for _, file := range files {
			detailUrl := dir.DetailUrl + "/" + file.Name()
			parent := dir.DetailUrl
			pathId := joinNotEmpty([]string{r.Root, r.RelativePath, r.Name, file.Name()}, "/")

			if file.IsDir() {
				dir.Children = append(dir.Children, Dir{Root: r.Root, Name: file.Name(), DetailUrl: detailUrl, ParentUrl: parent, PathId: pathId, ParentId: dir.PathId})
			} else {
				dir.Children = append(dir.Children, newMedia(r.Root, joinNotEmpty([]string{r.RelativePath, r.Name}, "/"), file.Name(), r.Host))
			}
		}
	}

	return dir, err
}

type Media struct {
	Root      string `json:"root"`
	Name      string `json:"name"`
	PathId    string `json:"pathId"`
	localPath string `json:"localPath"`
	ParentId  string `json:"parentId,omitempty"`

	ParentUrl string `json:"parentUrl,omitempty"`
	DetailUrl string `json:"detailUrl"`
	PlayUrl   string `json:"playUrl,omitempty"`
}

func newMedia(root string, relativePath string, name string, host string) Media {

	media := Media{Root: root, Name: name, localPath: joinNotEmpty([]string{roots[root], relativePath, name}, "/"), PathId: joinNotEmpty([]string{root, relativePath, name}, "/"), ParentId: joinNotEmpty([]string{root, relativePath}, "/")}

	parentUrl := url.URL{Scheme: "http", Host: host, Path: joinNotEmpty([]string{BROWSER_PREFIX + "/", root, relativePath}, "/")}
	media.ParentUrl = parentUrl.String()
	media.DetailUrl = parentUrl.String() + "/" + name

	publicPath := joinNotEmpty([]string{root, relativePath, name}, "/")
	play := url.URL{Scheme: "http", Host: host, Path: "/player/play", RawQuery: "media=" + url.QueryEscape(publicPath)}
	media.PlayUrl = play.String()

	return media

}
func newMediaFromRequest(r *PathRequest) Media {
	return newMedia(r.Root, r.RelativePath, r.Name, r.Host)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	glog.V(1).Infoln("Marshaled: ", string(response), "(payload=", payload, ")")

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		w.Write(response)
	}

	return err
}

func (m *Media) String() string {
	return fmt.Sprintf("%s (%s)", m.Name, m.localPath)
}
func (m *Dir) String() string {
	return fmt.Sprintf("%s (%s) - %d children", m.Name, m.localPath, len(m.Children))
}

// strings.Join, but without empty values
func joinNotEmpty(values []string, separator string) string {
	var res string
	for _, val := range values {
		if val != "" {
			if res != "" {
				res += "/"
			}

			res += val
		}
	}

	return res
}
