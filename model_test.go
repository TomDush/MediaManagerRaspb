package main

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/corbym/gocrest"
	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
	"github.com/golang/glog"
)

func TestNewPath(t *testing.T) {
	roots = map[string]Path{
		"foo": {Root: "foo", localPath: "/mnt/foo/data"},
		"bar": {Root: "bar", localPath: "/home/user1/data"},
	}
	type args struct {
		root string
		path string
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    Path
		wantErr bool
	}{
		{"create path with known root", args{"foo", "1/2/3", "4"}, Path{localPath: "/mnt/foo/data/1/2/3/4", Root: "foo", Name: "4", MiddlePath: "1/2/3"}, false},
		{"create path without middle", args{"bar", "", "42"}, Path{localPath: "/home/user1/data/42", Root: "bar", Name: "42", MiddlePath: ""}, false},
		{"create path with unknown root", args{"baz", "b", "az"}, Path{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := NewPath(tt.args.root, tt.args.path, tt.args.name); (err != nil) != tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPath() = %v, want %v, wantErr %v", got, tt.want, tt.wantErr)
			}
		})
	}
}

func TestPath_PathId(t *testing.T) {
	type fields struct {
		localPath  string
		Root       string
		MiddlePath string
		Name       string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"Full path with all elements", fields{"toto", "foo", "bar", "baz"}, "foo/bar/baz"},
		{"only root", fields{"toto", "foo", "", ""}, "foo"},
		{"only root and name", fields{"toto", "foo", "", "baz"}, "foo/baz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := &Path{
				localPath:  tt.fields.localPath,
				Root:       tt.fields.Root,
				MiddlePath: tt.fields.MiddlePath,
				Name:       tt.fields.Name,
			}
			if got := path.PathId(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PathId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_IsIndex(t *testing.T) {
	type fields struct {
		localPath  string
		Root       string
		MiddlePath string
		Name       string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"false: Full path with all elements", fields{"toto", "foo", "bar", "baz"}, false},
		{"false: root and name", fields{"toto", "foo", "", "baz"}, false},
		{"false: root only", fields{"toto", "foo", "", ""}, false},
		{"true: not even root", fields{"toto", "", "", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := &Path{
				localPath:  tt.fields.localPath,
				Root:       tt.fields.Root,
				MiddlePath: tt.fields.MiddlePath,
				Name:       tt.fields.Name,
			}
			if got := path.IsIndex(); got != tt.want {
				t.Errorf("path.IsIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Relative(t *testing.T) {
	roots = map[string]Path{
		"foo": {Root: "foo", localPath: "/mnt/foo/data"},
		"bar": {Root: "bar", localPath: "/home/user1/data"},
	}

	type fields struct {
		Root       string
		MiddlePath string
		Name       string
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Path
	}{
		{"Relative from full path", fields{"foo", "bar", "baz"}, args{"toto"}, Path{"/mnt/foo/data/bar/baz/toto", "foo", "bar/baz", "toto"}},
		{"Relative from only root and name", fields{"foo", "", "baz"}, args{"toto"}, Path{"/mnt/foo/data/baz/toto", "foo", "baz", "toto"}},
		{"Relative from only root", fields{"foo", "", ""}, args{"toto"}, Path{"/mnt/foo/data/toto", "foo", "", "toto"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := &Path{
				Root:       tt.fields.Root,
				MiddlePath: tt.fields.MiddlePath,
				Name:       tt.fields.Name,
			}
			if got := path.Relative(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("path.Relative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_ToFile(t *testing.T) {
	HasName := Applying(func(f File) string { return f.Path().Name }, "File.name")
	IsDir := Applying(func(f File) bool { return f.IsDir() }, "File.IsDir")
	HasChild := Applying(func(f File) []File {
		if dir, ok := f.(*Dir); ok {
			return dir.Children
		}
		return []File{}
	}, "File.Children")

	roots = map[string]Path{
		"wd": {Root: "wd", localPath: workingDir()},
	}

	type fields struct {
		MiddlePath string
		Name       string
	}
	type args struct {
		summarised bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		matcher *gocrest.Matcher
		wantErr bool
	}{
		{"working dir is a directory that contains model_test.go", fields{"", ""}, args{false}, is.AllOf(
			IsDir(is.True()), HasChild(AnyMatch(HasName(is.EqualTo("model_test.go"))))), false},
		{"working dir is a directory without children when summarised", fields{"", ""}, args{true}, is.AllOf(
			IsDir(is.True()), is.Not(HasChild(AnyMatch(HasName(is.EqualTo("model_test.go")))))), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, _ := NewPath("wd", tt.fields.MiddlePath, tt.fields.Name)
			got, err := path.ToFile(tt.args.summarised)
			if (err != nil) != tt.wantErr {
				t.Errorf("path.ToFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			then.AssertThat(t, got, tt.matcher)
		})
	}
}

func workingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		glog.Fatal(err)
	}
	return dir
}

// Map element before matching it
// ex:
//    WithLen := Applying(func(s string) int { return len(s) }, "string length")
//    then.AssertThat(t, "foo", WithLen(is.EqualTo(3))
func Applying(function interface{}, name string) func(*gocrest.Matcher) *gocrest.Matcher {
	funcValue := reflect.ValueOf(function)

	return func(delegate *gocrest.Matcher) *gocrest.Matcher {
		matcher := &gocrest.Matcher{
			Describe: name + " " + delegate.Describe,
		}

		matcher.Matches = func(actual interface{}) bool {
			if returned := funcValue.Call([]reflect.Value{reflect.ValueOf(actual)}); len(returned) > 0 {
				transformed := returned[0].Interface()
				matches := delegate.Matches(transformed)
				matcher.AppendActual("[" + name + "=")
				if delegate.Actual != "" {
					matcher.AppendActual(delegate.Actual + "]")
				} else {
					matcher.AppendActual(fmt.Sprint(transformed, "]"))
				}
				matcher.ReasonString = delegate.ReasonString
				return matches
			}

			glog.Fatalln("Applying function is expected to return 1 and  only 1 result.")
			return false
		}
		return matcher
	}
}

// Any element of the slice or Array match the given matcher
func AnyMatch(matcher *gocrest.Matcher) *gocrest.Matcher {
	any := new(gocrest.Matcher)
	any.Describe = fmt.Sprintf("contains 1 element where %s", matcher.Describe)
	any.Matches = func(actual interface{}) bool {
		any.Actual = fmt.Sprint(actual)
		if slice := reflect.ValueOf(actual); slice.Kind() == reflect.Slice {
			for i := 0; i < slice.Len(); i++ {
				if matcher.Matches(slice.Index(i).Interface()) {
					return true
				}
			}
		} else {
			const errorString = "is not a slice or array"
			any.AppendActual(errorString)
			any.ReasonString = fmt.Sprintf(errorString)
		}

		return false
	}

	return any
}

func TestPath_Ext(t *testing.T) {
	type fields struct {
		localPath  string
		Root       string
		MiddlePath string
		Name       string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"extension in the name is captured", fields{"/mnt/data/foo/bar.MP4", "data", "foo", "bar.mp4"}, "mp4"},
		{"extension in the name is captured, even if several dots", fields{"/mnt/data/foo.baz/bar.mp4", "data", "foo.baz", "bar.mp4"}, "mp4"},
		{"no extension give empty string", fields{"/mnt/data/foo/bar", "data", "foo", "bar"}, ""},
		{"simple root dir doesn't have extension", fields{"/mnt/data", "data", "", ""}, ""},
		{"simple root dir doesn't have extension (even if dot is present)", fields{"/mnt/data.foo", "data", "", ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := &Path{
				localPath:  tt.fields.localPath,
				Root:       tt.fields.Root,
				MiddlePath: tt.fields.MiddlePath,
				Name:       tt.fields.Name,
			}
			if got := path.Ext(); got != tt.want {
				t.Errorf("Path.Ext() = %v, want %v", got, tt.want)
			}
		})
	}
}
