package drum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	stepsInTrack  = 16
	playSoundMark = "x"
)

type Pattern struct {
	Version string
	Tempo   float32
	Tracks  []track
}

type track struct {
	ID    int
	Name  string
	Steps []byte
}

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var header struct {
		Splice   [6]byte  // 6 bytes
		FileSize int64    // 8 bytes
		Version  [32]byte // 32 bytes
	}

	err = binary.Read(file, binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal header from binary file")
	}

	trimmedVersion := string(bytes.Trim(header.Version[:], "\x00"))

	var tempo float32 // 4 bytes
	err = binary.Read(file, binary.LittleEndian, &tempo)

	var trackHeader struct {
		ID       byte
		WordSize int32
	}

	var tracks []track

	for {
		err = binary.Read(file, binary.BigEndian, &trackHeader)
		if err == io.EOF {
			break
		}

		trackName := make([]byte, trackHeader.WordSize)
		_, err := io.ReadFull(file, trackName)
		if err != nil {
			return nil, fmt.Errorf("Unable to read track name")
		}

		stepBytes := make([]byte, stepsInTrack)
		_, err = io.ReadFull(file, stepBytes)
		if err != nil {
			return nil, fmt.Errorf("Unable to read track steps")
		}

		steps := []byte(strings.Repeat("-", stepsInTrack))
		for k := range steps {
			if stepBytes[k] == 1 {
				steps[k] = 'x'
			}
		}

		track := track{
			ID:    int(trackHeader.ID),
			Name:  string(trackName),
			Steps: steps,
		}

		tracks = append(tracks, track)
	}

	pattern := Pattern{
		Version: trimmedVersion,
		Tempo:   tempo,
		Tracks:  tracks,
	}

	return &pattern, err
}

func (p *Pattern) String() string {
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
	trackBody := fmt.Sprintf("|%s|%s|%s|%s|\n", t.Steps[0:4], t.Steps[4:8], t.Steps[8:12], t.Steps[12:15])

	return trackHeader + trackBody
}
