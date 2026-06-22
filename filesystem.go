package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type filesystemEntry struct {
	name string
	path string
	kind string
	size int64
}

func listDirectoryEntries(root string) ([]filesystemEntry, error) {
	children, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("cannot list directory: %w", err)
	}

	entries := make([]filesystemEntry, 0, len(children))
	for _, child := range children {
		path := filepath.Join(root, child.Name())
		info, err := child.Info()
		if err != nil {
			return nil, fmt.Errorf("cannot inspect directory entry: %w", err)
		}
		size := info.Size()
		if info.IsDir() {
			size, err = directoryContentSize(path)
			if err != nil {
				return nil, fmt.Errorf("cannot measure directory entry: %w", err)
			}
		}
		entries = append(entries, filesystemEntry{
			name: child.Name(),
			path: path,
			kind: entryKind(info),
			size: size,
		})
	}
	return entries, nil
}

func treeEntries(root string) ([]filesystemEntry, error) {
	var entries []filesystemEntry
	directoryIndexes := make(map[string]int)

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}

		item := filesystemEntry{path: path, kind: entryKind(info)}
		if info.IsDir() {
			directoryIndexes[path] = len(entries)
		} else {
			item.size = info.Size()
		}
		entries = append(entries, item)

		if info.Mode().IsRegular() {
			for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
				if index, ok := directoryIndexes[parent]; ok {
					entries[index].size += info.Size()
				}
				if parent == root {
					break
				}
			}
		}
		return nil
	})
	return entries, err
}

func directoryContentSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func entryKind(info os.FileInfo) string {
	switch {
	case info.IsDir():
		return "directory"
	case info.Mode().IsRegular():
		return "file"
	case info.Mode()&os.ModeSymlink != 0:
		return "symlink"
	default:
		return "other"
	}
}
