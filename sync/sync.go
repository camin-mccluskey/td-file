package sync

import (
	"sync"

	"td-file/parser"

	"github.com/fsnotify/fsnotify"
)

type FileSynchronizer struct {
	Path     string
	ReloadCh chan struct{}
	SaveCh   chan []parser.Todo
	stopCh   chan struct{}
	mu       sync.Mutex
}

func NewFileSynchronizer(path string) *FileSynchronizer {
	return &FileSynchronizer{
		Path:     path,
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 1),
		stopCh:   make(chan struct{}),
	}
}

func (fs *FileSynchronizer) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := watcher.Add(fs.Path); err != nil {
		return err
	}
	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					select {
					case fs.ReloadCh <- struct{}{}:
					default:
					}
				}
			case <-fs.stopCh:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case todos := <-fs.SaveCh:
				fs.mu.Lock()
				parser.WriteTodosToFile(fs.Path, todos)
				fs.mu.Unlock()
			case <-fs.stopCh:
				return
			}
		}
	}()
	return nil
}

func (fs *FileSynchronizer) Stop() {
	close(fs.stopCh)
}
