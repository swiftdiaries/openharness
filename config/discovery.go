package config

import (
	"os"
	"path/filepath"
)

const (
	MaxInstructionFileChars  = 4000
	MaxTotalInstructionChars = 12000
)

type InstructionFile struct {
	Path    string
	Content string
}

func DiscoverInstructionFiles(startDir string) []InstructionFile {
	candidates := []string{"GHOSTFIN.md", ".ghostfin/GHOSTFIN.md", "GHOSTFIN.local.md"}
	var files []InstructionFile
	seen := map[string]bool{}
	totalChars := 0

	dir, _ := filepath.Abs(startDir)
	for {
		for _, candidate := range candidates {
			path := filepath.Join(dir, candidate)
			if seen[path] {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			seen[path] = true
			content := string(data)
			if len(content) > MaxInstructionFileChars {
				content = content[:MaxInstructionFileChars] + "\n[truncated]"
			}
			if totalChars+len(content) > MaxTotalInstructionChars {
				break
			}
			totalChars += len(content)
			files = append(files, InstructionFile{Path: path, Content: content})
		}
		if totalChars >= MaxTotalInstructionChars {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return files
}
