package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"github.com/golang/glog"
	"os"
	"sort"
)

var roots = make(map[string]Path)

// Load roots configuration
func ConfigureRoots() error {
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
		roots[r[0]] = Path{localPath: r[1], Root: r[0]}
	}

	return nil
}

type Path struct {
	localPath string

	Root       string `json:"root"`
	MiddlePath string `json:"-"`
	Name       string `json:"name"`
}

func NewPath(root string, path string, name string) (Path, error) {
	// TODO error when path doesn't exist
	empty := Path{}
	if root == "" && path == "" && name == "" {
		return empty, nil

	} else if roots[root] == empty {
		return empty, fmt.Errorf("invalid root: %s", root)
	}

	return Path{localPath: joinNotEmpty([]string{roots[root].localPath, path, name}, "/"), Root: root, MiddlePath: path, Name: name}, nil
}

// ID used from outside
func (path *Path) PathId() string {
	return joinNotEmpty([]string{path.Root, path.MiddlePath, path.Name}, "/")
}

// ID of parent directory used from outside
func (path *Path) ParentId() string {
	return joinNotEmpty([]string{path.Root, path.MiddlePath}, "/")
}
func (path *Path) DisplayName() string {
	name := path.Name
	if name == "" {
		name = path.Root
	}
	return name
}
func (path *Path) IsIndex() bool {
	return path.Root == ""
}
func (path *Path) Relative(name string) Path {
	// error ignored because root have already been validated on this path
	p, _ := NewPath(path.Root, joinNotEmpty([]string{path.MiddlePath, path.Name}, "/"), name)
	return p
}

// Extract and return file extension (after last dot, or file name if no dot)
func (path *Path) Ext() string {
	dots := strings.Split(path.Name, ".")
	if len(dots) == 1 {
		return ""
	}
	return strings.ToLower(dots[len(dots)-1])
}
func (path *Path) ToFile(summarised bool) (File, error) {
	if path.IsIndex() {
		// List available "roots"
		index := NewDir(*path)
		for _, root := range roots {
			index.Children = append(index.Children, NewDir(root))
		}

		return index, nil

	} else {
		stat, err := os.Stat(path.localPath)
		if err != nil {
			return nil, err
		}

		if stat.IsDir() {
			glog.V(1).Infoln("Browse directory ", path.localPath)
			dir := NewDir(*path)
			if !summarised {
				dir.loadChildren()
			}

			return dir, nil

		} else {
			glog.V(1).Infoln("Get media details: ", path.localPath)
			media := NewMedia(*path)
			return media, nil
		}
	}
}

type File interface {
	Path() *Path

	IsDir() bool
	Type() string
}
type FileBase struct {
	path Path
}

func (fileBase *FileBase) Path() *Path {
	return &fileBase.path
}
func (*FileBase) IsDir() bool {
	return false
}
func (fileBase *FileBase) String() string {
	return fmt.Sprintf("%s (%s)", fileBase.Path().Name, fileBase.Path().localPath)
}

type Dir struct {
	FileBase

	Children []File `json:"children,omitempty"`
}

func (*Dir) IsDir() bool {
	return true
}
func (*Dir) Type() string {
	return "dir"
}

// Simple constructor
func NewDir(path Path) *Dir {
	return &Dir{FileBase: FileBase{path: path}}
}

// Load children into structure
func (dir *Dir) loadChildren() error {
	files, err := ioutil.ReadDir(dir.path.localPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		filename := file.Name()
		glog.Info("File name is: ", filename)
		path := dir.path
		newPath := path.Relative(filename)
		f, err := newPath.ToFile(true)
		if err != nil {
			return err
		}
		dir.Children = append(dir.Children, f)
	}

	// And sort ny name
	sort.Sort(&dirSorter{dir.Children})

	return nil
}

// Utilities to sort files within a Dir
type dirSorter struct {
	files []File
}

// Len is part of sort.Interface.
func (s *dirSorter) Len() int {
	return len(s.files)
}

// Swap is part of sort.Interface.
func (s *dirSorter) Swap(i, j int) {
	s.files[i], s.files[j] = s.files[j], s.files[i]
}

// Less is part of sort.Interface.
func (s *dirSorter) Less(i, j int) bool {
	return strings.ToLower(s.files[i].Path().Name) < strings.ToLower(s.files[j].Path().Name)
}


type Media struct {
	FileBase
}

func (*Media) Type() string {
	return "media"
}

// Simple constructor
func NewMedia(path Path) *Media {
	return &Media{FileBase: FileBase{path: path}}
}
