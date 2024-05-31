package vfs

import (
	"io/fs"
	"os"
	"sort"
)

type MergeFS struct {
	// FileSystems is a list of file systems to merge.
	// The file systems are searched in the order they are listed.
	FileSystems []fs.FS
}

func NewMergeFS(fileSystems ...fs.FS) *MergeFS {
	return &MergeFS{FileSystems: fileSystems}
}

func (mfs *MergeFS) Open(name string) (fs.File, error) {
	for _, filesystem := range mfs.FileSystems {
		file, err := filesystem.Open(name)
		if err == nil {
			return file, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, fs.ErrNotExist
}

func (mfs *MergeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entriesMap := make(map[string]fs.DirEntry)
	for _, filesystem := range mfs.FileSystems {
		entries, err := fs.ReadDir(filesystem, name)
		if err == nil {
			for _, entry := range entries {
				if _, exists := entriesMap[entry.Name()]; !exists {
					entriesMap[entry.Name()] = entry
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var entries []fs.DirEntry
	for _, entry := range entriesMap {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}
