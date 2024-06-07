package revealjs

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server struct {
	port     int
	revealJS *RevealJS
}

func NewServer(port int, revealJS *RevealJS) *Server {
	return &Server{port, revealJS}
}

func (s *Server) Start() error {
	watcher, err := NewWatcher(s.revealJS.DataDirectory(), func() {
		// User may change config.yml. Reload it.
		s.revealJS.ReloadConfig()
	})
	if err != nil {
		return err
	}
	// TODO 終了処理いらないっけ？
	go func() {
		http.HandleFunc("/revision", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(watcher.Revision.Value))
		})

		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// Is index.html
			if req.URL.Path == "/" {
				// Generate index.html
				buf := &bytes.Buffer{}
				if err := s.revealJS.GenerateIndexHTML(buf, &HTMLGeneratorParams{
					HotReload: true,
					Revision:  &watcher.Revision.Value,
				}); err != nil {
					http.Error(w, "failed to generate index.html", http.StatusInternalServerError)
					return
				}
				http.ServeContent(w, req, "index.html", time.Now(), bytes.NewReader(buf.Bytes()))
				return
			}

			http.ServeFileFS(w, req, s.revealJS.FileSystem(), req.URL.Path)
		})
		log.Printf("Start server on http://localhost:%d", s.port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
		if err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()
	go watcher.Start()
	return nil
}
