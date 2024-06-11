package revealjs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
					log.Println(err)
					http.Error(w, "failed to generate index.html", http.StatusInternalServerError)
					return
				}
				http.ServeContent(w, req, "index.html", time.Now(), bytes.NewReader(buf.Bytes()))
				return
			}

			// If the file is markdown, remove the yaml header.
			if IsMarkdown(req.URL.Path) {
				file, err := s.revealJS.FileSystem().Open(req.URL.Path[1:]) // remove '/'
				if err != nil {
					log.Println(err)
					http.Error(w, "failed to open file", http.StatusInternalServerError)
					return
				}
				defer file.Close()
				b, err := io.ReadAll(file)
				if err != nil {
					http.Error(w, "failed to read file", http.StatusInternalServerError)
				}
				content := NewMarkdown(string(b)).WithoutYAMLHeader()
				http.ServeContent(w, req, req.URL.Path, time.Now(), strings.NewReader(content))
				return
			}

			// Other files
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
