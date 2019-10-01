package drum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// NOTE TO REVIEWER:
// Using package scope for types you don't want the consumer to instantiate
// themselves, always seems like a good pattern to me.
// In this case, I'm allowing the user to get a pattern back (from calling DecodeFile),
// but I don't want them creating their own.
// However, go-lint always complains about this being "confusing"
type pattern struct {
	Version string
	Tempo   float32
	Tracks  []track

	fileSize int64
}

type track struct {
	ID    int
	Name  string
	Steps []byte
}

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	pattern := pattern{}

	err = readHeader(file, &pattern)
	if err != nil {
		return nil, fmt.Errorf("Unable to read file header")
	}

	for {
		offset, err := file.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, fmt.Errorf("Unable to determine current seek position")
		}

		if offset > pattern.fileSize {
			break
		}

		err = readTrack(file, &pattern)
		if err != nil {
			return nil, fmt.Errorf("Unable to read track")
		}
	}

	return &pattern, nil
}

func readHeader(file io.Reader, p *pattern) error {
	var header struct {
		Splice   [6]byte
		FileSize int64
		Version  [32]byte
	}

	err := binary.Read(file, binary.BigEndian, &header)
	if err != nil {
		return fmt.Errorf("Unable to marshal header from binary file")
	}
	p.fileSize = header.FileSize

	err = binary.Read(file, binary.LittleEndian, &p.Tempo)
	if err != nil {
		return fmt.Errorf("Unable to read pattern tempo")
	}

	const NullCharacter = "\x00"
	p.Version = string(bytes.TrimRight(header.Version[:], NullCharacter))

	return nil
}

func readTrack(file io.Reader, p *pattern) error {
	var trackHeader struct {
		ID       byte
		WordSize int32
	}

	err := binary.Read(file, binary.BigEndian, &trackHeader)
	if err != nil {
		return fmt.Errorf("Unable to read track header")
	}

	trackName := make([]byte, trackHeader.WordSize)
	_, err = io.ReadFull(file, trackName)
	if err != nil {
		return fmt.Errorf("Unable to read track name")
	}

	const stepsInTrack = 16
	stepBytes := make([]byte, stepsInTrack)
	_, err = io.ReadFull(file, stepBytes)
	if err != nil {
		return fmt.Errorf("Unable to read track steps")
	}

	for k := range stepBytes {
		if stepBytes[k] == 1 {
			stepBytes[k] = 'x'
		} else {
			stepBytes[k] = '-'
		}
	}

	track := track{
		ID:    int(trackHeader.ID),
		Name:  string(trackName),
		Steps: stepBytes,
	}

	p.Tracks = append(p.Tracks, track)

	return nil
}

func (p *pattern) String() string {
	var result string

	result = fmt.Sprintf("Saved with HW Version: %v\n", p.Version)
	result += fmt.Sprintf("Tempo: %v\n", p.Tempo)

	for _, track := range p.Tracks {
		result += track.String()
	}

	return result
}

func (t track) String() string {
	trackHeader := fmt.Sprintf("(%v) %v\t", t.ID, t.Name)
	trackBody := fmt.Sprintf("|%s|%s|%s|%s|\n", t.Steps[0:4], t.Steps[4:8], t.Steps[8:12], t.Steps[12:16])

	return trackHeader + trackBody
}
