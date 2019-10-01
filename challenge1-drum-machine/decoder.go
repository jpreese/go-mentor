package drum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	stepsInTrack = 16
)

type Pattern struct {
	Version string
	Tempo   float32
	Tracks  []track
}

type track struct {
	id    int
	name  string
	steps [stepsInTrack]bool
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
		Id       byte
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

		var steps [stepsInTrack]bool
		for k := range stepBytes {
			steps[k] = stepBytes[k] == 1
		}

		track := track{
			id:    int(trackHeader.Id),
			name:  string(trackName),
			steps: steps,
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

	for k := range p.Tracks {
		result += fmt.Sprintf("%v", p.Tracks[k])
	}

	return result
}

func (t track) String() string {

	var result string
	for index, step := range t.steps {
		if index%5 == 0 {
			result += "|"
		}

		if step {
			result += "x"
		} else {
			result += "-"
		}

	}

	return fmt.Sprintf("%v\n", result)
}
