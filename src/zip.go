package main

import (
	"archive/zip"
	"bytes"
)

// ZipEntry is a single file to be placed into a zip archive.
type ZipEntry struct {
	Name    string
	Content []byte
}

func BuildZip(entries []ZipEntry, rootDir string) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)

	for _, entry := range entries {
		name := entry.Name
		if rootDir != "" {
			name = rootDir + "/" + name
		}

		w, err := writer.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(entry.Content); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
