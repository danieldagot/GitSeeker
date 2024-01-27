
# Go Directory Scanner

## Introduction
This Go project is a command-line tool designed to scan directories and identify Git repositories within them. It searches through the user's `Documents` and `Desktop` directories, skipping predefined directories (like `node_modules`, `dist`, etc.). The tool also features a readline interface for user interaction, allowing users to search and open projects in VS Code directly from the terminal.

## Requirements
- Go (1.x or later)
- Readline library (github.com/chzyer/readline)
- Access to the user's `Documents` and `Desktop` directories
- VS Code (for opening projects, optional)

## Installation

First, clone the repository to your local machine:

```sh
git clone [URL_of_Your_Repository]
cd [Your_Repository_Name]
```

Then build the project:

```sh
go build
```

This will create an executable in your project directory.

## Usage

Run the executable with or without verbose mode:

```sh
./[executable_name]             # Normal mode
./[executable_name] -ver        # Verbose mode
```

In the application:

1. Type a project name to search for it.
2. If a matching project is found, it will attempt to open it in VS Code.
3. Type `exit` to quit the application.

## Features

- Scans `Documents` and `Desktop` directories for Git repositories.
- Skips predefined directories to optimize the scanning process.
- Provides a readline interface with autocomplete for easy navigation.
- Opens selected projects in VS Code (if installed).
- Verbose mode for detailed logging during the scan.

## Note

This project is configured for macOS environments. Adjustments might be necessary for compatibility with other operating systems.
