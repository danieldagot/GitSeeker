// scanner.go - Repository scanning functionality
package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Directory represents a Git repository directory
type Directory struct {
	Name string
	Path string
}

type Scanner struct {
	config   Config
	verbose  bool
	maxDepth int
	ctx      context.Context
	cancel   context.CancelFunc
}

type ScanResult struct {
	Repositories []Directory
	Stats        ScanStats
	Error        error
}

type ScanStats struct {
	Duration       time.Duration
	FoldersScanned int
	ReposFound     int
	ErrorsIgnored  int
}

func NewScanner(config Config, verbose bool) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scanner{
		config:   config,
		verbose:  verbose,
		maxDepth: config.MaxDepth,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Scanner) Stop() {
	s.cancel()
}

func (s *Scanner) ScanRepositories() ScanResult {
	startTime := time.Now()
	var repos []Directory
	var mu sync.RWMutex
	var wg sync.WaitGroup

	stats := ScanStats{}

	// Use a worker pool to limit concurrent goroutines
	const maxWorkers = 10
	semaphore := make(chan struct{}, maxWorkers)

	for _, scanPath := range s.config.ScanPaths {
		if s.isContextCancelled() {
			break
		}

		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			pathRepos, pathStats := s.scanPath(path, 0)

			mu.Lock()
			repos = append(repos, pathRepos...)
			stats.FoldersScanned += pathStats.FoldersScanned
			stats.ReposFound += pathStats.ReposFound
			stats.ErrorsIgnored += pathStats.ErrorsIgnored
			mu.Unlock()
		}(scanPath)
	}

	wg.Wait()
	stats.Duration = time.Since(startTime)

	return ScanResult{
		Repositories: repos,
		Stats:        stats,
	}
}

func (s *Scanner) scanPath(path string, depth int) ([]Directory, ScanStats) {
	if depth > s.maxDepth || s.isContextCancelled() {
		return nil, ScanStats{}
	}

	stats := ScanStats{}
	var repos []Directory

	// Check if directory exists and is accessible
	info, err := os.Stat(path)
	if err != nil {
		if s.verbose {
			fmt.Printf("Warning: Cannot access %s: %v\n", path, err)
		}
		stats.ErrorsIgnored++
		return repos, stats
	}

	if !info.IsDir() {
		return repos, stats
	}

	stats.FoldersScanned++

	// Check if this is a git repository
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		repoName := filepath.Base(path)
		repos = append(repos, Directory{
			Name: repoName,
			Path: path,
		})
		stats.ReposFound++
		if s.verbose {
			fmt.Printf("Found repository: %s at %s\n", repoName, path)
		}
		return repos, stats // Don't recurse into git repos
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		if s.verbose {
			fmt.Printf("Warning: Cannot read directory %s: %v\n", path, err)
		}
		stats.ErrorsIgnored++
		return repos, stats
	}

	var wg sync.WaitGroup
	var mu sync.RWMutex

	for _, entry := range entries {
		if !entry.IsDir() || s.isContextCancelled() {
			continue
		}

		dirName := entry.Name()

		// Skip hidden directories unless configured to include them
		if !s.config.IncludeHidden && strings.HasPrefix(dirName, ".") {
			continue
		}

		// Skip directories in the skip list
		if s.shouldSkipDir(dirName) {
			continue
		}

		subPath := filepath.Join(path, dirName)

		wg.Add(1)
		go func(subPath string) {
			defer wg.Done()
			subRepos, subStats := s.scanPath(subPath, depth+1)

			mu.Lock()
			repos = append(repos, subRepos...)
			stats.FoldersScanned += subStats.FoldersScanned
			stats.ReposFound += subStats.ReposFound
			stats.ErrorsIgnored += subStats.ErrorsIgnored
			mu.Unlock()
		}(subPath)
	}

	wg.Wait()
	return repos, stats
}

func (s *Scanner) shouldSkipDir(dirName string) bool {
	for _, skipDir := range s.config.SkipDirs {
		if dirName == skipDir {
			return true
		}
	}
	return false
}

func (s *Scanner) isContextCancelled() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}
