package main

import (
	"github.com/golang/glog"
	"flag"
	"time"
	"github.com/gorilla/mux"

	"fmt"
	"net/http"
)

var mmConfig *MmConfig

// Run with ./rasbmm -stderrthreshold=INFO -v=3 for debug
func main() {
	var err error

	mmConfig = new(MmConfig)
	flag.StringVar(&mmConfig.www, "www", ".", "the directory to serve files from. Defaults to the current dir")
	flag.IntVar(&mmConfig.port, "port", 8080, "port on which server is started. Defaults to 8080")
	flag.StringVar(&mmConfig.roots, "roots", "", "(required) coma separated list of media directories")

	flag.Parse()
	err = mmConfig.IsValid()

	if err == nil {
		glog.Infoln("Bootstraping MediaManager designed for Raspberries...")

		r := mux.NewRouter()
		StaticController(r)
		err = BrowserController(r)

		if err == nil {
			srv := &http.Server{
				Handler:      r,
				Addr:         mmConfig.HostAndPort(),
				WriteTimeout: 15 * time.Second,
				ReadTimeout:  15 * time.Second,
			}
			srv.ListenAndServe()
		}

	}

	if err != nil {
		glog.Fatal("Can not start server: " + err.Error())
	}
}

func StaticController(r *mux.Router) {
	glog.V(1).Infoln("Add static controller endpoints")
	r.HandleFunc("/health", healthCheck)

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(mmConfig.www))))
}
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "{\"status\": \"%s\"}", "OK")
}

type MmConfig struct {
	port  int
	roots string
	www   string
}

func (c *MmConfig) IsValid() error {
	var err error
	if len(c.roots) == 0 {
		err = fmt.Errorf("'roots' must be specified (-root=<coma sperated list>)")
	}

	glog.V(1).Infoln("Configuration loaded: ", mmConfig.String())
	return err
}
func (c *MmConfig) HostAndPort() string {
	return fmt.Sprintf(":%d", c.port)
}
func (c *MmConfig) String() string {
	return fmt.Sprintf("MmCOnfig[port=%d, www=%s, roots=%s, HostAndPort=%s]", c.port, c.www, c.roots, c.HostAndPort())
}

func GetMmConfig() *MmConfig {
	return mmConfig
}
