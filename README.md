# GitSeeker

A fast, configurable command-line tool to find and open Git repositories in your favorite editor.

## Features

- 🔍 **Fast Repository Discovery** - Recursively scans configured directories for Git repositories
- ⚡ **Smart Caching** - Cache results for faster subsequent runs
- 🎯 **Interactive Mode** - Tab completion and command history
- ⚙️ **Configurable** - JSON configuration file for customization
- 🚀 **Multiple Editors** - Support for VS Code, Sublime Text, Vim, and more
- 🎨 **Colored Output** - Beautiful terminal interface with colors
- 🔧 **Graceful Interruption** - Handle Ctrl+C gracefully during scans
- 📊 **Scan Statistics** - Detailed performance metrics
- 💡 **Smart Suggestions** - Suggests similar project names on typos

## Installation

### From Source
```bash
git clone https://github.com/yourusername/GitSeeker.git
cd GitSeeker
go build -o gs
```

### Using Go Install
```bash
go install github.com/yourusername/GitSeeker@latest
```

## Usage

### Basic Usage
```bash
# Start interactive mode
./gs

# Open a project directly
./gs my-project

# List all repositories
./gs -list

# Use cached results (faster)
./gs -cache

# Verbose output
./gs -v

# Use different editor
./gs -editor vim

# Show configuration
./gs -config
```

### Command Line Flags

| Flag | Short | Description |
|------|-------|-------------|
| `-list` | `-l` | List all Git repositories and exit |
| `-verbose` | `-v` | Enable verbose logging |
| `-cache` | | Use cached results (faster startup) |
| `-refresh` | | Force refresh cache |
| `-config` | | Show configuration file location and exit |
| `-editor` | | Override default editor (e.g., 'code', 'subl', 'vim') |
| `-depth` | | Maximum scan depth (0 = use config default) |
| `-install` | | Install binary to ~/.local/bin and zsh completions |
| `-uninstall` | | Remove installed binary and zsh completions |

### Interactive Mode Commands

| Command | Description |
|---------|-------------|
| `help`, `h` | Show available commands |
| `list` | List all repositories |
| `config` | Show configuration information |
| `refresh` | Refresh repository list |
| `exit`, `quit`, `q` | Exit the program |
| `<project-name>` | Open project in configured editor |

## Zsh Completion

### Quick Install

The easiest way to install is with the built-in flag:
```bash
./gs -install
```

This will:
1. Copy the binary to `~/.local/bin/gs`
2. Install zsh completions to `~/.zsh/completions/_gs`
3. Detect **Oh My Zsh** and configure completions correctly (adds `fpath` before `source $ZSH/oh-my-zsh.sh`)
4. For non-Oh My Zsh setups, offer to add `fpath`, `compinit`, and `autoload` to your `~/.zshrc`
5. Offer to add `~/.local/bin` to your `PATH` if it's missing

Each step asks for confirmation before modifying your `~/.zshrc`.

### Manual Install

If you prefer to set things up yourself:

1. Copy the completion file:
```bash
mkdir -p ~/.zsh/completions
cp completions/_gs ~/.zsh/completions/_gs
```

2. Add to your `~/.zshrc`:

**With Oh My Zsh** — add this line *before* `source $ZSH/oh-my-zsh.sh`:
```bash
fpath=(~/.zsh/completions $fpath)
```

**Without Oh My Zsh** — add these lines:
```bash
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit
compinit
```

3. Reload your shell:
```bash
source ~/.zshrc
```

### Uninstall

```bash
gs -uninstall
```

This removes the binary and completion file. You may also want to remove the `fpath` and `PATH` entries from your `~/.zshrc`.

Completion uses cached repositories when available. If you add new projects, run `gs -refresh`.

## Configuration

GitSeeker uses a JSON configuration file located at `~/.gitseeker/config.json`.

### Default Configuration
```json
{
  "scan_paths": [
    "/Users/username/Documents",
    "/Users/username/Desktop",
    "/Users/username/Projects"
  ],
  "skip_dirs": [
    "node_modules", "dist", "build", "target", "vendor", ".git",
    ".vscode", ".idea", "bin", "obj", "out", "tmp", "temp",
    "logs", "cache", ".next", ".nuxt", "coverage"
  ],
  "editor": "code",
  "max_depth": 5,
  "include_hidden": false
}
```

### Configuration Options

- **scan_paths**: Directories to scan for Git repositories
- **skip_dirs**: Directory names to skip during scanning
- **editor**: Default editor command (code, subl, vim, etc.)
- **max_depth**: Maximum directory depth to scan
- **include_hidden**: Whether to include hidden directories (starting with .)

## Performance

- **Concurrent Scanning**: Uses goroutines for parallel directory scanning
- **Worker Pool**: Limits concurrent operations to prevent resource exhaustion
- **Smart Caching**: 24-hour cache validity for faster subsequent runs
- **Configurable Depth**: Limit scan depth to improve performance
- **Skip Lists**: Extensive skip lists to avoid scanning unnecessary directories

## Examples

### Basic Interactive Session
```
$ ./gs
Found 15 Git repositories. Type a project name and press Tab for autocompletion.
Type 'help' for available commands.

GitSeeker> my-pr[TAB]
my-project   my-prototype

GitSeeker> my-project
Opening code at: /Users/username/Documents/my-project
Successfully opened project: my-project
```

### Listing Repositories
```
$ ./gs -list
Found 15 Git repositories:
  1. awesome-app
     /Users/username/Documents/awesome-app
  2. my-project
     /Users/username/Documents/my-project
  ...
```

### Using Cache
```
$ ./gs -cache -v
Using cached results...
Found 15 Git repositories. Type a project name and press Tab for autocompletion.
```

## Building

### Development Build
```bash
go build -o gs
```

### Install (binary + zsh completion)
```bash
make install
```

### Install to custom prefix
```bash
make install PREFIX=$HOME/.local
```

### Production Build
```bash
go build -ldflags "-s -w" -o gs
```

### Cross-Platform Builds
```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o gitseeker.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o gs-mac

# Linux
GOOS=linux GOARCH=amd64 go build -o gs-linux
```

## Dependencies

- [github.com/chzyer/readline](https://github.com/chzyer/readline) - Interactive readline functionality

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v2.0.0 (Current)
- ✨ Added configuration file support
- ⚡ Implemented smart caching
- 🎯 Enhanced interactive mode with command history
- 🔧 Added graceful shutdown handling
- 📊 Added scan statistics
- 💡 Added smart suggestions for typos
- 🎨 Improved UI with colors and better formatting
- 🚀 Better error handling and user feedback
- ⚙️ Configurable editor support
- 🔍 Performance optimizations with worker pools

### v1.0.0
- 🔍 Basic Git repository scanning
- 📁 Interactive project selection
- 🚀 VS Code integration
- ⌨️ Tab completion
