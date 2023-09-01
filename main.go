package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

func transformRules(url string) string {
	// Modify the URL if it starts with arxiv.org/abs/
	if strings.HasPrefix(url, "http://arxiv.org/abs/") || strings.HasPrefix(url, "https://arxiv.org/abs/") {
		url = strings.Replace(url, "/abs/", "/pdf/", 1) + ".pdf"
	}
	return url
}

func downloadFile(url string, dest string) (string, error) {
	url = transformRules(url)

	// Extract filename from URL
	_, filename := filepath.Split(url)

	// Check if file already exists in the destination
	fullPath := filepath.Join(dest, filename)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		return filename, nil // Return if file already exists
	}

	// Get the data from the URL
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return filename, err
}

func processMarkdownFile(path string, root string) error {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Use regex to find the patterns
	r := regexp.MustCompile(`!\(indexer\)(http[^\s]+)`)
	matches := r.FindAllSubmatch(content, -1)

	for _, match := range matches {
		if len(match) == 2 {
			url := string(match[1])
			filename, err := downloadFile(url, filepath.Join(root, "attachments"))
			if err != nil {
				return err
			}
			replacement := []byte(`[indexer/pdf](/attachments/` + filename + `)`)
			content = bytes.Replace(content, match[0], replacement, 1)
		}
	}

	// Write back to the file
	return os.WriteFile(path, content, 0644)
}

func daemonize() {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		log.Fatal(err)
	}

	defer devNull.Close()

	attr := &syscall.ProcAttr{
		Files: []uintptr{devNull.Fd(), devNull.Fd(), devNull.Fd()},
	}

	// First fork
	pid, err := syscall.ForkExec(os.Args[0], os.Args, attr)
	if err != nil {
		fmt.Println("Error during first fork: ", err)
		os.Exit(1)
	}

	if pid > 0 {
		// We're in the parent process of the first fork, exit.
		os.Exit(0)
	}

	// Create a new session
	_, err = syscall.Setsid()
	if err != nil {
		log.Fatalf("Error creating new session: %v", err)
	}

	// Second fork to ensure the child is not a session leader
	pid, err = syscall.ForkExec(os.Args[0], os.Args, attr)
	if err != nil {
		fmt.Println("Error during second fork: ", err)
		os.Exit(1)
	}

	if pid > 0 {
		// We're in the parent process of the second fork, exit.
		os.Exit(0)
	}
}

func argparse() (string, bool) {
	var root string // Replace with your root directory
	var asDaemon bool
	flag.BoolVar(&asDaemon, "d", false, "Set this process run backgroud as daemon (only for Linux and Macos)")
	flag.StringVar(&root, "r", "", "Scan the directory")
	flag.Parse()
	if root == "" {
		fmt.Printf("Please use -r to specify the root directory to scan markdown files.")
	}
	if asDaemon {
		daemonize()
	}

	root, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}
	return root, asDaemon
}

func main() {
	root, asDaemon := argparse()
	_ = os.Mkdir(filepath.Join(root, "attachments"), os.ModePerm)

	if asDaemon {
		// If asDaemon is true, watch for file changes
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}
		defer watcher.Close()

		// Initial scan and process, also add directories to the watcher
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if !d.IsDir() && filepath.Ext(path) == ".md" {
				return processMarkdownFile(path, root)
			}
			return watcher.Add(path)
		})

		done := make(chan bool)

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
						if filepath.Ext(event.Name) == ".md" {
							err := processMarkdownFile(event.Name, root)
							if err != nil {
								log.Println("Error processing markdown:", err)
							}
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("Watcher error:", err)
				}
			}
		}()

		<-done

	} else {
		// If asDaemon is false, just do a one-time processing
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && filepath.Ext(path) == ".md" {
				return processMarkdownFile(path, root)
			}
			return nil
		})

		if err != nil {
			panic(err)
		}
	}
}
