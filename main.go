// main.go - Main package for GitSeeker
package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"workspace/utils"

	"github.com/chzyer/readline"
)

//go:embed completions/_gs
var embeddedCompletionFile embed.FS

// Directory struct to hold the name and path
type Directory = utils.Directory
type ScanStats = utils.ScanStats
type Config = utils.Config
type Scanner = utils.Scanner

var (
	verboseFlag    bool
	listFlag       bool
	configFlag     bool
	cacheFlag      bool
	refreshFlag    bool
	completionFlag bool
	installFlag    bool
	uninstallFlag  bool
	editorFlag     string
	maxDepthFlag   int
)

func main() {
	flag.BoolVar(&verboseFlag, "ver", false, "Enable verbose logging")
	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose logging (short)")
	flag.BoolVar(&listFlag, "l", false, "List all Git repositories and exit")
	flag.BoolVar(&listFlag, "list", false, "List all Git repositories and exit")
	flag.BoolVar(&configFlag, "config", false, "Show configuration file location and exit")
	flag.BoolVar(&cacheFlag, "cache", false, "Use cached results (faster startup)")
	flag.BoolVar(&refreshFlag, "refresh", false, "Force refresh cache")
	flag.BoolVar(&completionFlag, "completion-list", false, "Print completion candidates and exit")
	flag.BoolVar(&installFlag, "install", false, "Install binary to ~/.local/bin and zsh completions")
	flag.BoolVar(&uninstallFlag, "uninstall", false, "Remove installed binary and zsh completions")
	flag.StringVar(&editorFlag, "editor", "", "Override default editor (e.g., 'code', 'subl', 'vim')")
	flag.IntVar(&maxDepthFlag, "depth", 0, "Maximum scan depth (0 = use config default)")
	flag.Parse()

	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		log.Printf("Warning: Failed to load config, using defaults: %v", err)
		config = utils.GetDefaultConfig()
	}

	// Override config with command line flags
	if editorFlag != "" {
		config.Editor = editorFlag
	}
	if maxDepthFlag > 0 {
		config.MaxDepth = maxDepthFlag
	}

	// Handle config flag
	if configFlag {
		showConfigInfo(config)
		return
	}

	if installFlag {
		installCompletions()
		return
	}

	if uninstallFlag {
		uninstall()
		return
	}

	if completionFlag {
		printCompletionList(config)
		return
	}

	if args := flag.Args(); len(args) > 0 {
		openProjectByArg(args[0], config)
		return
	}

	// Setup graceful shutdown
	scanner := utils.NewScanner(config, verboseFlag)
	setupGracefulShutdown(scanner)

	// Scan for repositories
	var result utils.ScanResult
	if cacheFlag && !refreshFlag {
		if cached := utils.LoadCache(); cached != nil && len(cached.Repositories) > 0 {
			result = *cached
			if verboseFlag {
				fmt.Println("Using cached results...")
			}
		} else {
			result = scanner.ScanRepositories()
			utils.SaveCache(&result)
		}
	} else {
		result = scanner.ScanRepositories()
		if cacheFlag {
			utils.SaveCache(&result)
		}
	}

	if result.Error != nil {
		log.Fatal("Scan failed:", result.Error)
	}

	if verboseFlag {
		printScanStats(result.Stats)
	}

	if len(result.Repositories) == 0 {
		fmt.Println("No Git repositories found in configured scan paths.")
		fmt.Printf("Scan paths: %v\n", config.ScanPaths)
		fmt.Println("Run with -config to see configuration file location.")
		return
	}

	// Sort repositories by name for better UX
	sort.Slice(result.Repositories, func(i, j int) bool {
		return strings.ToLower(result.Repositories[i].Name) < strings.ToLower(result.Repositories[j].Name)
	})

	// Handle list flag
	if listFlag {
		fmt.Printf("Found %d Git repositories:\n", len(result.Repositories))
		for i, dir := range result.Repositories {
			fmt.Printf("%3d. %s\n     %s\n", i+1, dir.Name, dir.Path)
		}
		return
	}

	// Start interactive mode
	startInteractiveMode(result.Repositories, config)
}

func printCompletionList(config utils.Config) {
	repos, err := loadRepositoriesForCli(config)
	if err != nil {
		return
	}

	sort.Slice(repos, func(i, j int) bool {
		return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name)
	})

	for _, dir := range repos {
		fmt.Println(dir.Name)
	}
}

func loadRepositoriesForCli(config utils.Config) ([]Directory, error) {
	if cached := utils.LoadCache(); cached != nil && len(cached.Repositories) > 0 {
		return cached.Repositories, nil
	}

	scanner := utils.NewScanner(config, verboseFlag)
	result := scanner.ScanRepositories()
	if result.Error != nil {
		return nil, result.Error
	}
	_ = utils.SaveCache(&result)
	return result.Repositories, nil
}

func installCompletions() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}

	installBinary(homeDir)
	installZshCompletions(homeDir)
}

func installBinary(homeDir string) {
	binDir := filepath.Join(homeDir, ".local", "bin")
	destBin := filepath.Join(binDir, "gs")

	self, err := os.Executable()
	if err != nil {
		fmt.Printf("Error: could not determine executable path: %v\n", err)
		os.Exit(1)
	}
	self, _ = filepath.EvalSymlinks(self)

	if existing, err := filepath.EvalSymlinks(destBin); err == nil && existing == self {
		fmt.Printf("Binary already installed at %s\n", destBin)
	} else {
		if err := os.MkdirAll(binDir, 0755); err != nil {
			fmt.Printf("Error: could not create directory %s: %v\n", binDir, err)
			os.Exit(1)
		}

		srcData, err := os.ReadFile(self)
		if err != nil {
			fmt.Printf("Error: could not read binary: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(destBin, srcData, 0755); err != nil {
			fmt.Printf("Error: could not write binary to %s: %v\n", destBin, err)
			os.Exit(1)
		}

		fmt.Printf("Installed binary to %s\n", destBin)
	}

	checkPathContains(homeDir, binDir)
}

func askUser(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func checkPathContains(homeDir, dir string) {
	pathEnv := os.Getenv("PATH")
	for _, p := range filepath.SplitList(pathEnv) {
		if p == dir {
			return
		}
	}

	fmt.Printf("\nWarning: %s is not in your PATH.\n", dir)

	zshrcPath := filepath.Join(homeDir, ".zshrc")
	exportLine := fmt.Sprintf("export PATH=\"%s:$PATH\"", dir)

	if data, err := os.ReadFile(zshrcPath); err == nil {
		if strings.Contains(string(data), dir) {
			fmt.Println("It appears to already be in your ~/.zshrc. Restart your shell to apply.")
			return
		}
	}

	if askUser("Would you like to add it to your ~/.zshrc?") {
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error: could not open ~/.zshrc: %v\n", err)
			fmt.Printf("Add this manually to your ~/.zshrc:\n  %s\n", exportLine)
			return
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + exportLine + "\n"); err != nil {
			fmt.Printf("Error: could not write to ~/.zshrc: %v\n", err)
			return
		}
		fmt.Println("Added PATH entry to ~/.zshrc.")
		fmt.Println("Run 'source ~/.zshrc' or restart your shell to apply.")
	} else {
		fmt.Printf("Add this to your ~/.zshrc manually:\n  %s\n", exportLine)
	}
}

func installZshCompletions(homeDir string) {
	destDir := filepath.Join(homeDir, ".zsh", "completions")
	destFile := filepath.Join(destDir, "_gs")

	if _, err := os.Stat(destFile); err == nil {
		fmt.Printf("Zsh completions already installed at %s\n", destFile)
		checkZshrcFpath(homeDir)
		return
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Printf("Error: could not create directory %s: %v\n", destDir, err)
		os.Exit(1)
	}

	data, err := embeddedCompletionFile.ReadFile("completions/_gs")
	if err != nil {
		fmt.Printf("Error: could not read embedded completion file: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		fmt.Printf("Error: could not write completion file to %s: %v\n", destFile, err)
		os.Exit(1)
	}

	fmt.Printf("Installed zsh completions to %s\n", destFile)
	checkZshrcFpath(homeDir)
}

func checkZshrcFpath(homeDir string) {
	zshrcPath := filepath.Join(homeDir, ".zshrc")
	fpathLine := "fpath=(~/.zsh/completions $fpath)"

	data, err := os.ReadFile(zshrcPath)
	if err != nil {
		fmt.Println("\nNo ~/.zshrc found.")
		if askUser("Would you like to create one with completion support?") {
			content := fpathLine + "\nautoload -Uz compinit\ncompinit\n"
			if err := os.WriteFile(zshrcPath, []byte(content), 0644); err != nil {
				fmt.Printf("Error: could not create ~/.zshrc: %v\n", err)
				return
			}
			fmt.Println("Created ~/.zshrc with completion support.")
		} else {
			fmt.Println("Add the following to your shell config manually:")
			fmt.Printf("  %s\n", fpathLine)
			fmt.Println("  autoload -Uz compinit")
			fmt.Println("  compinit")
		}
		return
	}

	content := string(data)

	if strings.Contains(content, ".zsh/completions") {
		fmt.Println("Your ~/.zshrc already references ~/.zsh/completions.")
		fmt.Println("Run 'source ~/.zshrc' or restart your shell to pick up changes.")
		return
	}

	isOhMyZsh := strings.Contains(content, "oh-my-zsh.sh")

	if isOhMyZsh {
		fmt.Println("\nOh My Zsh detected. The fpath line needs to be added before 'source $ZSH/oh-my-zsh.sh'.")
		if askUser("Would you like to add it automatically?") {
			newContent := strings.Replace(content, "source $ZSH/oh-my-zsh.sh", fpathLine+"\nsource $ZSH/oh-my-zsh.sh", 1)
			if err := os.WriteFile(zshrcPath, []byte(newContent), 0644); err != nil {
				fmt.Printf("Error: could not update ~/.zshrc: %v\n", err)
				return
			}
			fmt.Println("Added fpath entry to ~/.zshrc (before Oh My Zsh source).")
			fmt.Println("Run 'source ~/.zshrc' or restart your shell to apply.")
		} else {
			fmt.Printf("Add this line before 'source $ZSH/oh-my-zsh.sh' in your ~/.zshrc:\n  %s\n", fpathLine)
		}
	} else {
		fmt.Println("\nCompletions need fpath and compinit configuration in ~/.zshrc.")
		if askUser("Would you like to add it automatically?") {
			f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("Error: could not open ~/.zshrc: %v\n", err)
				return
			}
			defer f.Close()

			lines := "\n" + fpathLine + "\nautoload -Uz compinit\ncompinit\n"
			if _, err := f.WriteString(lines); err != nil {
				fmt.Printf("Error: could not write to ~/.zshrc: %v\n", err)
				return
			}
			fmt.Println("Added completion support to ~/.zshrc.")
			fmt.Println("Run 'source ~/.zshrc' or restart your shell to apply.")
		} else {
			fmt.Println("Add the following to your ~/.zshrc manually:")
			fmt.Printf("  %s\n", fpathLine)
			fmt.Println("  autoload -Uz compinit")
			fmt.Println("  compinit")
		}
	}
}

func uninstall() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}

	removed := 0

	binPath := filepath.Join(homeDir, ".local", "bin", "gs")
	if _, err := os.Stat(binPath); err == nil {
		if err := os.Remove(binPath); err != nil {
			fmt.Printf("Error: could not remove %s: %v\n", binPath, err)
		} else {
			fmt.Printf("Removed %s\n", binPath)
			removed++
		}
	} else {
		fmt.Printf("Binary not found at %s (skipped)\n", binPath)
	}

	compPath := filepath.Join(homeDir, ".zsh", "completions", "_gs")
	if _, err := os.Stat(compPath); err == nil {
		if err := os.Remove(compPath); err != nil {
			fmt.Printf("Error: could not remove %s: %v\n", compPath, err)
		} else {
			fmt.Printf("Removed %s\n", compPath)
			removed++
		}
	} else {
		fmt.Printf("Completions not found at %s (skipped)\n", compPath)
	}

	if removed > 0 {
		fmt.Println("\nUninstall complete. You may also want to remove the PATH and fpath entries from ~/.zshrc.")
	} else {
		fmt.Println("\nNothing to uninstall.")
	}
}

func openProjectByArg(input string, config utils.Config) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	repos, err := loadRepositoriesForCli(config)
	if err != nil {
		fmt.Printf("Error loading repositories: %v\n", err)
		return
	}

	for _, dir := range repos {
		if dir.Name == input {
			fmt.Printf("Opening %s at: %s\n", config.Editor, dir.Path)
			if err := openInEditor(dir.Path, config.Editor); err != nil {
				fmt.Printf("Error opening %s: %v\n", config.Editor, err)
				fmt.Printf("Make sure %s is installed and available in PATH\n", config.Editor)
				return
			}
			fmt.Printf("Successfully opened project: %s\n", input)
			return
		}
	}

	fmt.Printf("Project '%s' not found.\n", input)
	suggestions := findSimilarNames(input, repos)
	if len(suggestions) > 0 {
		fmt.Printf("Did you mean: %s\n", strings.Join(suggestions, ", "))
	}
	fmt.Printf("Use Tab for autocompletion or type 'gs -list' to see all projects.\n")
}

func startInteractiveMode(repos []Directory, config utils.Config) {
	// Create directory map and completer items
	dirMap := make(map[string]string)
	var completerItems []readline.PrefixCompleterInterface

	for _, dir := range repos {
		dirMap[dir.Name] = dir.Path
		completerItems = append(completerItems, readline.PcItem(dir.Name))
	}

	// Add command completions
	commandCompleter := readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("list"),
		readline.PcItem("config"),
		readline.PcItem("refresh"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
	)

	// Combine project and command completers
	allCompleters := append(completerItems, commandCompleter.Children...)
	completer := readline.NewPrefixCompleter(allCompleters...)

	const (
		colorGreen  = "\033[32m"
		colorReset  = "\033[0m"
		colorRed    = "\033[31m"
		colorYellow = "\033[33m"
		colorBlue   = "\033[34m"
		colorCyan   = "\033[36m"
		colorGray   = "\033[90m"
	)

	// Setup readline with enhanced completer
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          colorGreen + "GitSeeker> " + colorReset,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		HistoryFile:     getHistoryFile(),
	})

	if err != nil {
		log.Fatal("Failed to initialize readline:", err)
	}
	defer rl.Close()

	fmt.Printf("%sFound %d Git repositories.%s Type a project name and press Tab for autocompletion.\n",
		colorBlue, len(repos), colorReset)
	fmt.Printf("Type '%shelp%s' for available commands.\n", colorYellow, colorReset)
	fmt.Printf("%sPress Tab twice for suggestions, or type a few letters and press Tab!%s\n\n", colorCyan, colorReset)

	// Interactive loop with enhanced input handling
	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// Show live suggestions for partial input
		if len(trimmedLine) > 0 && !isCommand(trimmedLine) {
			if _, exists := dirMap[trimmedLine]; !exists {
				suggestions := findSimilarNames(trimmedLine, repos)
				fmt.Printf("suggestions: %s\n", suggestions)
				if len(suggestions) > 0 {
					fmt.Printf("%sMatching repositories:%s %s%s%s\n",
						colorGray, colorReset, colorCyan, strings.Join(suggestions, ", "), colorReset)
					if len(suggestions) <= 1 {
						fmt.Printf("%sPress Tab to autocomplete or type the full name%s\n", colorGray, colorReset)
					} else {
						fmt.Printf("%sPress Tab to autocomplete or be more specific%s\n", colorGray, colorReset)
					}
					continue
				}
			}
		}

		switch trimmedLine {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return

		case "help", "h":
			printHelp()

		case "list":
			fmt.Printf("\n%sAvailable repositories:%s\n", colorBlue, colorReset)
			for i, dir := range repos {
				fmt.Printf("%3d. %s\n     %s%s%s\n", i+1, dir.Name, colorYellow, dir.Path, colorReset)
			}
			fmt.Println()

		case "config":
			showConfigInfo(config)

		case "refresh":
			fmt.Println("Refreshing repository list...")
			scanner := utils.NewScanner(config, verboseFlag)
			result := scanner.ScanRepositories()
			if result.Error != nil {
				fmt.Printf("%sError refreshing: %v%s\n", colorRed, result.Error, colorReset)
			} else {
				repos = result.Repositories
				sort.Slice(repos, func(i, j int) bool {
					return strings.ToLower(repos[i].Name) < strings.ToLower(repos[j].Name)
				})

				// Update completer and map
				dirMap = make(map[string]string)
				completerItems = nil
				for _, dir := range repos {
					dirMap[dir.Name] = dir.Path
					completerItems = append(completerItems, readline.PcItem(dir.Name))
				}

				fmt.Printf("%sFound %d repositories%s\n", colorGreen, len(repos), colorReset)
				utils.SaveCache(&result)
			}

		default:
			if path, ok := dirMap[trimmedLine]; ok {
				fmt.Printf("Opening %s at: %s\n", config.Editor, path)
				if err := openInEditor(path, config.Editor); err != nil {
					fmt.Printf("%sError opening %s: %v%s\n", colorRed, config.Editor, err, colorReset)
					fmt.Printf("Make sure %s is installed and available in PATH\n", config.Editor)
				} else {
					fmt.Printf("%sSuccessfully opened project: %s%s\n", colorGreen, trimmedLine, colorReset)
					return
				}
			} else {
				fmt.Printf("%sProject '%s' not found.%s\n", colorRed, trimmedLine, colorReset)

				// Suggest similar names
				suggestions := findSimilarNames(trimmedLine, repos)
				if len(suggestions) > 0 {
					fmt.Printf("%sDid you mean:%s %s%s%s\n", colorYellow, colorReset, colorGreen, strings.Join(suggestions, ", "), colorReset)
				}
				fmt.Printf("Use Tab for autocompletion or type '%slist%s' to see all projects.\n", colorYellow, colorReset)
			}
		}
	}
}

func openInEditor(path, editor string) error {
	cmd := exec.Command(editor, path)
	return cmd.Start()
}

func printHelp() {
	const (
		colorYellow = "\033[33m"
		colorReset  = "\033[0m"
		colorGreen  = "\033[32m"
	)

	fmt.Printf("\n%sAvailable Commands:%s\n", colorYellow, colorReset)
	fmt.Printf("  %shelp, h%s        - Show this help\n", colorGreen, colorReset)
	fmt.Printf("  %slist%s           - List all repositories\n", colorGreen, colorReset)
	fmt.Printf("  %sconfig%s         - Show configuration info\n", colorGreen, colorReset)
	fmt.Printf("  %srefresh%s        - Refresh repository list\n", colorGreen, colorReset)
	fmt.Printf("  %sexit, quit, q%s  - Exit the program\n", colorGreen, colorReset)
	fmt.Printf("  %s<project-name>%s - Open project in configured editor\n", colorGreen, colorReset)
	fmt.Printf("\n%sUse Tab for autocompletion of project names.%s\n\n", colorYellow, colorReset)
}

func printScanStats(stats ScanStats) {
	fmt.Printf("Scan completed in: %v\n", stats.Duration)
	fmt.Printf("Folders scanned: %d\n", stats.FoldersScanned)
	fmt.Printf("Repositories found: %d\n", stats.ReposFound)
	if stats.ErrorsIgnored > 0 {
		fmt.Printf("Warnings/errors ignored: %d\n", stats.ErrorsIgnored)
	}
	fmt.Println()
}

func showConfigInfo(config Config) {
	homeDir, _ := os.UserHomeDir()
	configPath := fmt.Sprintf("%s/.gitseeker/config.json", homeDir)

	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Printf("Editor: %s\n", config.Editor)
	fmt.Printf("Max depth: %d\n", config.MaxDepth)
	fmt.Printf("Include hidden: %t\n", config.IncludeHidden)
	fmt.Printf("Scan paths: %v\n", config.ScanPaths)
	fmt.Printf("Skip directories: %v\n", config.SkipDirs)
}

func setupGracefulShutdown(scanner *Scanner) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nReceived interrupt signal, stopping scan...")
		scanner.Stop()
		os.Exit(0)
	}()
}

func getHistoryFile() string {
	homeDir, _ := os.UserHomeDir()
	historyDir := fmt.Sprintf("%s/.gitseeker", homeDir)
	os.MkdirAll(historyDir, 0755)
	return fmt.Sprintf("%s/history", historyDir)
}

func findSimilarNames(input string, repos []Directory) []string {
	var suggestions []string
	input = strings.ToLower(input)

	// First, try exact prefix matches
	for _, repo := range repos {
		repoName := strings.ToLower(repo.Name)
		if strings.HasPrefix(repoName, input) {
			suggestions = append(suggestions, repo.Name)
		}
	}

	// If we have enough suggestions, return them
	if len(suggestions) >= 3 {
		return suggestions[:3]
	}

	// Then try contains matches (current behavior)
	for _, repo := range repos {
		repoName := strings.ToLower(repo.Name)
		if strings.Contains(repoName, input) || strings.Contains(input, repoName) {
			// Avoid duplicates
			found := false
			for _, existing := range suggestions {
				if existing == repo.Name {
					found = true
					break
				}
			}
			if !found {
				suggestions = append(suggestions, repo.Name)
				if len(suggestions) >= 5 { // Increased limit for better fuzzy matching
					break
				}
			}
		}
	}

	// Finally, try fuzzy matching by individual characters
	if len(suggestions) < 3 {
		for _, repo := range repos {
			repoName := strings.ToLower(repo.Name)
			if fuzzyMatch(input, repoName) {
				// Avoid duplicates
				found := false
				for _, existing := range suggestions {
					if existing == repo.Name {
						found = true
						break
					}
				}
				if !found {
					suggestions = append(suggestions, repo.Name)
					if len(suggestions) >= 5 {
						break
					}
				}
			}
		}
	}

	return suggestions
}

// fuzzyMatch checks if all characters in input appear in order in target
func fuzzyMatch(input, target string) bool {
	if len(input) == 0 {
		return true
	}
	if len(input) > len(target) {
		return false
	}

	inputIndex := 0
	for _, char := range target {
		if inputIndex < len(input) && rune(input[inputIndex]) == char {
			inputIndex++
		}
	}

	return inputIndex == len(input)
}

// Helper function to check if input is a command
func isCommand(input string) bool {
	commands := []string{"help", "h", "list", "config", "refresh", "exit", "quit", "q"}
	for _, cmd := range commands {
		if input == cmd {
			return true
		}
	}
	return false
}
