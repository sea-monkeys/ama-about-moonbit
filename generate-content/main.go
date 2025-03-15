package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Options for the program
type Options struct {
	rootDir       string
	includeExts   []string
	excludeExts   []string
	structureFile string
	contentFile   string
	summaryFile   string
}

// Main function
func main() {
	// Command line options definition
	rootDir := flag.String("dir", ".", "Root directory to analyze")
	includeExtsStr := flag.String("include", "md,go,mbt", "Extensions to include (comma separated)")
	excludeExtsStr := flag.String("exclude", "html,css", "Extensions to exclude (comma separated)")
	structureFile := flag.String("structure", "directory-structure.txt", "Output file for directory structure")
	contentFile := flag.String("content", "content.txt", "Output file for file contents")
	summaryFile := flag.String("summary", "summary.txt", "Output file for statistics summary")

	flag.Parse()

	// Preparing options
	options := Options{
		rootDir:       *rootDir,
		includeExts:   normalizeExtensions(strings.Split(*includeExtsStr, ",")),
		excludeExts:   normalizeExtensions(strings.Split(*excludeExtsStr, ",")),
		structureFile: *structureFile,
		contentFile:   *contentFile,
		summaryFile:   *summaryFile,
	}

	// Check if root directory exists
	if _, err := os.Stat(options.rootDir); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist\n", options.rootDir)
		os.Exit(1)
	}

	// Display options
	fmt.Printf("Analyzing directory: %s\n", options.rootDir)
	fmt.Printf("Included extensions: %v\n", options.includeExts)
	fmt.Printf("Excluded extensions: %v\n", options.excludeExts)
	fmt.Printf("Structure file: %s\n", options.structureFile)
	fmt.Printf("Content file: %s\n", options.contentFile)
	fmt.Printf("Summary file: %s\n", options.summaryFile)

	// Analyze directory
	files, err := walkDirectory(options)
	if err != nil {
		fmt.Printf("Error analyzing directory: %v\n", err)
		os.Exit(1)
	}

	// Generate structure file
	if err := generateStructureFile(files, options); err != nil {
		fmt.Printf("Error generating structure file: %v\n", err)
		os.Exit(1)
	}

	// Collect statistics while generating content file
	stats, err := generateContentFile(files, options)
	if err != nil {
		fmt.Printf("Error generating content file: %v\n", err)
		os.Exit(1)
	}

	// Generate summary file
	if err := generateSummaryFile(stats, options); err != nil {
		fmt.Printf("Error generating summary file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Processing completed successfully!")
	
	// Display statistics summary in console
	fmt.Println("\nStatistics Summary:")
	fmt.Printf("Total files processed: %d\n", stats.TotalFiles)
	fmt.Printf("Total file size: %.2f KB\n", float64(stats.TotalSize)/1024.0)
	fmt.Printf("Average file size: %.2f KB\n", stats.AverageFileSize/1024.0)
	fmt.Printf("Total tokens: %d\n", stats.TotalTokens)
	fmt.Printf("Average tokens per file: %.2f\n", stats.AverageTokens)
}

// Normalize file extensions
func normalizeExtensions(exts []string) []string {
	result := make([]string, 0, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		// Ensure extension starts with a dot
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		result = append(result, ext)
	}
	return result
}

// Structure to represent a file or directory
type FileInfo struct {
	Path     string
	IsDir    bool
	Children []*FileInfo
	Size     int64  // File size in bytes
	Content  string // File content (used for token counting)
}

// Structure to store statistics
type Statistics struct {
	TotalFiles      int     // Total number of processed files
	TotalSize       int64   // Total size of files in bytes
	AverageFileSize float64 // Average file size in bytes
	TotalTokens     int     // Total number of tokens
	AverageTokens   float64 // Average number of tokens per file
}

// Recursively walk through the directory
func walkDirectory(options Options) ([]*FileInfo, error) {
	var filteredFiles []*FileInfo

	// Recursive walk function
	err := filepath.Walk(options.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// If it's a file, check its extension
		if !info.IsDir() {
			ext := filepath.Ext(path)
			
			// Check if file should be included based on extensions
			includeFile := false
			if len(options.includeExts) == 0 {
				includeFile = true // If no extensions specified, include all
			} else {
				for _, includeExt := range options.includeExts {
					if strings.EqualFold(ext, includeExt) {
						includeFile = true
						break
					}
				}
			}

			// Check if file should be excluded based on extensions
			for _, excludeExt := range options.excludeExts {
				if strings.EqualFold(ext, excludeExt) {
					includeFile = false
					break
				}
			}

			// If file should be included, add it to the list
			if includeFile {
				// Read file content for statistics
				content, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("unable to read file %s: %v", path, err)
				}
				
				filteredFiles = append(filteredFiles, &FileInfo{
					Path:    path,
					IsDir:   false,
					Size:    info.Size(),
					Content: string(content),
				})
			}
		}

		return nil
	})

	return filteredFiles, err
}

// Generate directory structure file with filtered files
func generateStructureFile(files []*FileInfo, options Options) error {
	// Create structure file
	file, err := os.Create(options.structureFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Title
	fmt.Fprintln(file, "Directory structure:")

	// Structure to keep track of processed directories
	dirs := make(map[string]bool)
	
	// Sort files by path for cleaner display
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Create tree structure
	rootPath := options.rootDir
	if rootPath == "." {
		var err error
		rootPath, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	rootName := filepath.Base(rootPath)
	fmt.Fprintf(file, "└── %s/\n", rootName)

	// For each filtered file
	for _, fileInfo := range files {
		// Relative path from root directory
		relPath, err := filepath.Rel(rootPath, fileInfo.Path)
		if err != nil {
			return err
		}

		// Split path into components
		components := strings.Split(relPath, string(filepath.Separator))
		
		// Display tree structure for this file
		for i := 0; i < len(components); i++ {
			// Build path up to this level
			currentPath := filepath.Join(components[:i+1]...)
			
			// Check if it's a directory or file
			isLastComponent := i == len(components)-1
			isDir := !isLastComponent
			
			// If it's a directory that hasn't been displayed
			if isDir && !dirs[currentPath] {
				dirs[currentPath] = true
				prefix := strings.Repeat("│   ", i) + "├── "
				fmt.Fprintf(file, "    %s%s/\n", prefix, components[i])
			} else if isLastComponent {
				// If it's the final file
				prefix := strings.Repeat("│   ", i) + "├── "
				fmt.Fprintf(file, "    %s%s\n", prefix, components[i])
			}
		}
	}

	return nil
}

// Generate content file with all filtered files and collect statistics
func generateContentFile(files []*FileInfo, options Options) (Statistics, error) {
	// Create content file
	contentFile, err := os.Create(options.contentFile)
	if err != nil {
		return Statistics{}, err
	}
	defer contentFile.Close()

	// Initialize statistics
	stats := Statistics{
		TotalFiles: len(files),
	}

	// For each filtered file
	for i, fileInfo := range files {
		// Separator
		if i > 0 {
			fmt.Fprintln(contentFile)
		}
		
		// Header for the file
		fmt.Fprintln(contentFile, "================================================")
		fmt.Fprintf(contentFile, "File %d: %s\n", i+1, fileInfo.Path)
		fmt.Fprintln(contentFile, "================================================")
		
		// Write content
		fmt.Fprintln(contentFile, fileInfo.Content)
		
		// Collect statistics
		stats.TotalSize += fileInfo.Size
		
		// Count tokens (words) in content
		tokens := countTokens(fileInfo.Content)
		stats.TotalTokens += tokens
	}

	// Calculate averages
	if stats.TotalFiles > 0 {
		stats.AverageFileSize = float64(stats.TotalSize) / float64(stats.TotalFiles)
		stats.AverageTokens = float64(stats.TotalTokens) / float64(stats.TotalFiles)
	}

	return stats, nil
}

// Count the number of tokens (words) in a text
func countTokens(text string) int {
	// Split text into words (tokens) using spaces and punctuation as separators
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	
	// Filter empty tokens
	var validWords []string
	for _, word := range words {
		if word != "" {
			validWords = append(validWords, word)
		}
	}
	
	return len(validWords)
}

// Generate summary file with statistics
func generateSummaryFile(stats Statistics, options Options) error {
	// Create summary file
	summaryFile, err := os.Create(options.summaryFile)
	if err != nil {
		return err
	}
	defer summaryFile.Close()

	// Write statistics
	fmt.Fprintln(summaryFile, "Statistics Summary")
	fmt.Fprintln(summaryFile, "=================")
	fmt.Fprintf(summaryFile, "Total files processed: %d\n", stats.TotalFiles)
	fmt.Fprintf(summaryFile, "Total file size: %.2f KB (%.2f MB)\n", 
		float64(stats.TotalSize)/1024.0, float64(stats.TotalSize)/(1024.0*1024.0))
	fmt.Fprintf(summaryFile, "Average file size: %.2f KB\n", stats.AverageFileSize/1024.0)
	fmt.Fprintf(summaryFile, "Total tokens: %d\n", stats.TotalTokens)
	fmt.Fprintf(summaryFile, "Average tokens per file: %.2f\n", stats.AverageTokens)

	return nil
}