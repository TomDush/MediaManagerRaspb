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

func BrowserController(r *mux.Router) error {
	glog.V(1).Infoln("Registring Browser Controller")

	config := GetMmConfig()
	rootConfig := strings.Split(config.roots, ",")
	if len(rootConfig) == 0 {
		return fmt.Errorf("Media roots must be required (-root key1:/path,key2:/path/2) ")
	}
	for i := 0; i < len(rootConfig); i++ {
		r := strings.Split(rootConfig[i], ":")
		glog.V(1).Infoln("Parsing ", r)
		if len(r) != 2 {
			return fmt.Errorf("roots configuration invalid '%s', it must be name=path", r)
		}
		roots[r[0]] = r[1]
	}

	r.PathPrefix("/browser/").HandlerFunc(ShowMedia)
	//mediaRouter := r.PathPrefix("/media/paths")
	//mediaRouter.HandleFunc("/media/paths", FindAllRoots)
	//r.HandleFunc("/media/paths", FindAllRoots)

	glog.Infoln("Browser Controller config loaded. Roots are:", roots)
	return nil
}

func ShowMedia(w http.ResponseWriter, r *http.Request) {
	filesystemPath, root, relativePath := resolveRequestedFile(r)

	parentPath := ""
	if strings.Contains(relativePath, "/") {
		parentPath = relativePath[:strings.LastIndex(relativePath, "/")]
	}
	parentUrl := url.URL{Scheme: "http", Host: r.Host, Path: "/browser/" + root + "/" + parentPath}

	var elem interface{}

	stat, err := os.Stat(filesystemPath)
	if err == nil {
		if stat.IsDir() {
			elem, err = newDir(root, relativePath, filesystemPath, parentUrl, r)

		} else {
			elem = newMedia(root, relativePath, parentUrl, r)
		}

		if err == nil {
			err = respondWithJSON(w, 200, elem)
		}
	}

	if err != nil {
		glog.Warning("Can't process path [", root, "]/", relativePath, ": ", err.Error())
		respondWithJSON(w, 500, map[string]string{"error": err.Error()})
	}
}
func newMedia(root string, relativePath string, parentUrl url.URL, r *http.Request) Media {
	media := Media{Root: root, Path: relativePath, Parent: parentUrl.String()}
	glog.V(1).Infoln("Requested media: ", media.String())

	play := url.URL{Scheme: "http", Host: r.Host, Path: "/player/play", RawQuery: "media=" + url.QueryEscape(fmt.Sprintf("%s/%s", root, relativePath))}
	media.Play = play.String()

	return media
}

func newDir(root string, relativePath string, filesystemPath string, parentUrl url.URL, r *http.Request) (dir Dir, err error) {
	dir = Dir{Root: root, Path: relativePath, filesystemPath: filesystemPath, Parent: parentUrl.String()}

	files, err := ioutil.ReadDir(dir.filesystemPath)

	if err == nil {
		for _, file := range files {
			detailUrl := url.URL{Scheme: "http", Host: r.Host, Path: "/browser/" + root + "/" + relativePath + "/" + file.Name()}
			parent := url.URL{Scheme: "http", Host: r.Host, Path: "/browser/" + root + "/" + relativePath}

			if file.IsDir() {
				dir.Children = append(dir.Children, Dir{Root: root, Path: relativePath + "/" + file.Name(), Details: detailUrl.String(), Parent: parent.String()})
			} else {
				dir.Children = append(dir.Children, Media{Root: root, Path: relativePath + "/" + file.Name(), Details: detailUrl.String(), Parent: parent.String()})
			}
		}
	}

	return dir, err
}

func resolveRequestedFile(r *http.Request) (string, string, string) {
	name := strings.TrimPrefix(r.URL.Path, "/browser/")
	glog.V(2).Infoln("Browsing to path: ", name)
	var relativePath string
	path := strings.Split(name, "/")
	for i := 1; i < len(path); i++ {
		relativePath += "/" + path[i]
	}
	root := path[0]

	return roots[root] + relativePath, strings.Trim(root, "/"), strings.Trim(relativePath, "/")
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

type Dir struct {
	Root           string
	Path           string
	filesystemPath string

	Parent  string
	Details string

	Children []interface{}
}

type Media struct {
	Root string
	Path string
	filesystemPath string

	Parent  string
	Details string
	Play string
}

func (m *Media) String() string {
	return fmt.Sprintf("%s [root=%s]", m.Path, m.Root)
}
func (m *Dir) String() string {
	return fmt.Sprintf("%s [root=%s, %d children]", m.filesystemPath, m.Root, len(m.Children))
}
