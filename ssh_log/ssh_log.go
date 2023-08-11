package ssh_log

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

const authFilePath = "/var/log/auth.log"

type Handler struct {
	filepath   string
	TimeFormat string
	watcher    *fsnotify.Watcher
}

type Message struct {
	LastLine string
	Time     time.Time
}

func New() *Handler {
	return &Handler{
		TimeFormat: "Jan 2 15:04:05 ",
	}
}

func (h *Handler) Close() error {
	return h.watcher.Close()
}

func (h *Handler) Start() (chan Message, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "NewWatcher")
	}

	h.watcher = watcher
	watcher.Add(authFilePath)

	var messageChan = make(chan Message)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					close(messageChan)
					h.Close()
					return
				}

				// Deal with log rotation
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("Log file was rotated")
					h.watcher.Remove(event.Name)
					h.watcher.Add(authFilePath)
				}

				if event.Op&fsnotify.Write != fsnotify.Write {
					continue
				}

				FilePath, err := filepath.Abs(event.Name)
				if err != nil {
					log.Printf("Failed to get absolute path: %s\n", err.Error())
					continue
				}

				// send only if the file is auth file
				if FilePath != authFilePath {
					continue
				}

				line, err := h.readAuthLastLine()
				if err != nil {
					log.Printf("Failed to read last line: %s\n", err.Error())
					continue
				}

				host, err := os.Hostname()
				if err != nil {
					log.Printf("Failed to get hostname: %s\n", err.Error())
					continue
				}

				var tlines = strings.Split(line, host)
				if len(tlines) < 2 {
					continue
				}

				printTime, err := time.Parse(h.TimeFormat, tlines[0])
				if err != nil {
					continue
				}

				messageChan <- Message{
					LastLine: strings.Join(tlines[1:], host),
					Time:     printTime,
				}

			case err, ok := <-watcher.Errors:
				log.Printf("Error at waching files: %v\n", err.Error())
				if !ok {
					close(messageChan)
					h.Close()
					return
				}
			}
		}
	}()

	return messageChan, nil
}

func (h *Handler) readAuthLastLine() (line string, err error) {
	const bufSize = 500

	file, err := os.Open(authFilePath)
	if err != nil {
		return "", errors.Wrap(err, "OpenAuthFile")
	}
	defer file.Close()

	file.Seek(-1*(bufSize+1), 2)

	var scanner = bufio.NewScanner(file)

	// get last line
	for scanner.Scan() {
		line = scanner.Text()
	}

	return line, nil
}
