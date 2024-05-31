package revealjs

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	revealjs *RevealJS
	Revision *Revision
}

type Revision struct {
	Value string
}

func (r *Revision) update() {
	r.Value = time.Now().String()
}

func NewWatcher(revealjs *RevealJS) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	rev := &Revision{}
	rev.update()
	return &Watcher{w, revealjs, rev}, err
}

func (w *Watcher) Start() {
	w.watcher.Add(w.revealjs.dataDirectory)
	filepath.Walk(w.revealjs.dataDirectory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			w.watcher.Add(path)
		}
		return nil
	})
	w.watcher.Add(w.revealjs.indexTemplate)
	for evt := range w.watcher.Events {
		op := evt.Op
		if op&fsnotify.Create != 0 {
			if s, err := os.Stat(evt.Name); !os.IsNotExist(err) && s.IsDir() {
				w.watcher.Add(evt.Name)
			}
			w.notifyUpdate()
		} else if op&fsnotify.Remove != 0 {
			w.watcher.Remove(evt.Name)
			w.notifyUpdate()
		} else if op&fsnotify.Write != 0 {
			w.notifyUpdate()
		}
	}
}

func (w *Watcher) notifyUpdate() {
	log.Println("Data directory updated.")
	w.Revision.update()
}
