package drum

import (
	"fmt"
	"os"
)

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	p := pattern{}

	err = p.readHeader(file)
	if err != nil {
		return nil, fmt.Errorf("Unable to read file header")
	}

	for {
		offset, err := file.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, fmt.Errorf("Unable to determine current seek position")
		}

		if offset > p.fileSize {
			break
		}

		err = p.readTrack(file)
		if err != nil {
			return nil, fmt.Errorf("Unable to read track")
		}
	}

	return &p, nil
}
