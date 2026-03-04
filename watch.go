package sex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type WatchEvent struct {
	Data []byte
	Err  error
}

type fsWatcher struct {
	source string
	events chan WatchEvent
	ctx    context.Context
	cancel context.CancelFunc
	inner  *fsnotify.Watcher
}

func newFSWatcher(source string) (*fsWatcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &fsWatcher{
		source: source,
		events: make(chan WatchEvent, 8),
		ctx:    ctx,
		cancel: cancel,
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		return nil, err
	}
	w.inner = watcher

	if err := w.addSource(source); err != nil {
		watcher.Close()
		cancel()
		return nil, err
	}

	go w.loop()
	return w, nil
}

func (w *fsWatcher) Events() <-chan WatchEvent {
	return w.events
}

func (w *fsWatcher) Close() {
	w.cancel()
}

func (w *fsWatcher) loop() {
	defer func() {
		w.inner.Close()
		close(w.events)
	}()
	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.inner.Events:
			if !ok {
				return
			}
			if !w.isSourceEvent(event.Name) {
				continue
			}
			if event.Has(fsnotify.Chmod) {
				continue
			}
			data, err := os.ReadFile(w.source)
			if err != nil {
				w.events <- WatchEvent{Err: err}
				return
			}
			w.events <- WatchEvent{Data: data}
		case err, ok := <-w.inner.Errors:
			if !ok {
				return
			}
			w.events <- WatchEvent{Err: err}
			return
		}
	}
}

func (w *fsWatcher) addSource(source string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("source is directory: %s", source)
	}
	dir := filepath.Dir(source)
	return w.inner.Add(dir)
}

func (w *fsWatcher) isSourceEvent(name string) bool {
	return filepath.Clean(name) == filepath.Clean(w.source)
}
