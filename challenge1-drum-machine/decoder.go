package drum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type track struct {
	ID    int
	Name  string
	Steps []byte
}

func (t track) String() string {
	trackHeader := fmt.Sprintf("(%v) %v\t", t.ID, t.Name)
	trackBody := fmt.Sprintf("|%s|%s|%s|%s|\n", t.Steps[0:4], t.Steps[4:8], t.Steps[8:12], t.Steps[12:16])

	return trackHeader + trackBody
}

// Pattern represents a decoded drum file
type Pattern struct {
	Version string
	Tempo   float32
	Tracks  []track

	fileSize int64
}

func (p *Pattern) readHeader(file io.Reader) error {
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

	// We use binary.LittleEndian here because the pattern file stores
	// the tempo value in LittleEndian.
	err = binary.Read(file, binary.LittleEndian, &p.Tempo)
	if err != nil {
		return fmt.Errorf("Unable to read pattern tempo")
	}

	const NullCharacter = "\x00"
	p.Version = string(bytes.TrimRight(header.Version[:], NullCharacter))

	return nil
}

func (p *Pattern) readTrack(file io.Reader) error {
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

func (p *Pattern) String() string {
	result := fmt.Sprintf("Saved with HW Version: %v\n", p.Version)
	result += fmt.Sprintf("Tempo: %v\n", p.Tempo)

	for _, track := range p.Tracks {
		result += track.String()
	}

	return result
}
