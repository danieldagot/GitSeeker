package main

import (
	"flag"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/chzyer/readline"
)

// Directory struct to hold the name and path
type Directory struct {
	Name string
	Path string
}

var verboseFlag bool

func main() {
	flag.BoolVar(&verboseFlag, "ver", false, "Enable verbose logging")
	flag.Parse()

	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	var folderCount, projectCount int
	startTime := time.Now()

	documentsDir := filepath.Join(currentUser.HomeDir, "Documents")
	desktopDir := filepath.Join(currentUser.HomeDir, "Desktop")

	var gitDirs []Directory
	var wg sync.WaitGroup
	dirChan := make(chan Directory)

	// Goroutine to collect directories
	go func() {
		for dir := range dirChan {
			gitDirs = append(gitDirs, dir)
		}
	}()

	// Function to scan directories
	scanDir := func(path string, info os.FileInfo, err error) error {
		if info != nil && info.IsDir() {
			dirName := info.Name()
			// List of directories to skip
            skipDirs := []string{"node_modules", "src", "dist", "deploy_node_modules", ".serverless", ".github", "config", "features", "build", "bin", "lib", "logs", "tmp", "temp", "env", "venv", ".vscode", ".idea", "public", "utils", ".esbuild", "settings", "secrets", "dagrams", "export", "images", "data", "test", "tests", "doc", "docs", "distros", "demo", "demos", "examples", "backup", "scripts", "assets", "archive", "installers", "locale", "install", "logs", "packages", "resources", "themes", "translations", "uploads", "videos", "webroot"}
			folderCount++ // Increment folder count

			// Check if the current directory is in the list of directories to skip
			for _, skipDir := range skipDirs {
				if dirName == skipDir {
					return filepath.SkipDir
				}
			}
			if verboseFlag {
				println("Scanning directory:", path) // Conditionally print
			}
			if info.Name() == ".git" {
				// Create a Directory struct and send it over the channel
				dir := Directory{
					Name: filepath.Base(filepath.Dir(path)),
					Path: filepath.Dir(path),
				}
				projectCount++ // Increment project count
				dirChan <- dir
				return filepath.SkipDir
			}
		}
		return nil
	}

	// Scan both directories
	scanAndProcessDirectory(documentsDir, &wg, dirChan, scanDir)
	scanAndProcessDirectory(desktopDir, &wg, dirChan, scanDir)

	// Wait for all goroutines to complete
	wg.Wait()
	close(dirChan)
	endTime := time.Now()
	scanDuration := endTime.Sub(startTime)

	if verboseFlag {
		println("Scanning completed in:", scanDuration.Milliseconds(), "milliseconds")
		println("Total folders scanned:", folderCount)
		println("Total projects found:", projectCount)
	}

	// Map to hold directory name to path mapping
	dirMap := make(map[string]string)
	var completerItems []readline.PrefixCompleterInterface

	for _, dir := range gitDirs {
		dirMap[dir.Name] = dir.Path
		completerItems = append(completerItems, readline.PcItem(dir.Name))
	}

	// Create the completer with the collected items
	completer := readline.NewPrefixCompleter(completerItems...)
	const (
		colorGreen = "\033[32m"
		colorReset = "\033[0m"
	)

	// Setup readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          colorGreen + "Enter project name: " + colorReset,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// Readline loop
	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "exit" {
			break
		} else if path, ok := dirMap[trimmedLine]; ok {
			println("Full path:", path)
			cmd := exec.Command("code", path)
			err := cmd.Start()
			if err != nil {
				println("Error opening VS Code:", err)
			} else {
				println("Opened VS Code at path:", path)
				break
			}
		} else {
			println("You entered:", trimmedLine)
		}
	}
}

func scanAndProcessDirectory(directoryPath string, wg *sync.WaitGroup, dirChan chan<- Directory, scanDirFn filepath.WalkFunc) {
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			go func(entry os.DirEntry) {
				defer wg.Done()
				subDirPath := filepath.Join(directoryPath, entry.Name())
				filepath.Walk(subDirPath, scanDirFn)
			}(entry)
		}
	}
}
