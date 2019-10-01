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

func (p *pattern) readHeader(file io.Reader) error {
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

	// NOTE TO REVIEWER:
	// It may seem a little verbose to store \x00 into its own const
	// but I personally am a fan of avoiding magic strings wherever possible.
	// Even though the value is only used once, it should be easier on the
	// reader to identify what is going on.
	const NullCharacter = "\x00"
	p.Version = string(bytes.TrimRight(header.Version[:], NullCharacter))

	return nil
}

func (p *pattern) readTrack(file io.Reader) error {
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

	// NOTE TO REVIEWER:
	// I wavered back and forth as to where to put this logic, and ultimately
	// decided to put it during the Decode() operation as I wanted the call to
	// .String() to be snappier. I imagine there would be more read operations
	// once the file has been decoded.
	// However, depending upon the intended use of the .Steps property in the API
	// it may be more beneficial to leave them as binary values if the consumer needs
	// them in that format to make musical sounds?
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
