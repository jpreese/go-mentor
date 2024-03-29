package drum

import (
	"fmt"
	"os"
)

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var p Pattern

	if err := p.readHeader(file); err != nil {
		return nil, fmt.Errorf("unable to read file header: %w", err)
	}

	for {
		offset, err := file.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, fmt.Errorf("unable to determine current seek position: %w", err)
		}

		if offset > p.fileSize {
			break
		}

		if err := p.readTrack(file); err != nil {
			return nil, fmt.Errorf("unable to read track: %w", err)
		}
	}

	return &p, nil
}
