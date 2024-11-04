package bencode

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// Unmarshal takes a byte slice of Bencode data and returns the decoded value.
func Unmarshal(data []byte) (any, error) {
	reader := bytes.NewReader(data)
	return unmarshalValue(reader)
}

// unmarshalValue determines the type of the value and calls the appropriate unmarshal function.
func unmarshalValue(r io.Reader) (any, error) {
	ch, err := readByte(r)
	if err != nil {
		return nil, err
	}

	switch ch {
	case 'i':
		return unmarshalInt(r)
	case 'l':
		return unmarshalList(r)
	case 'd':
		return unmarshalDict(r)
	default:
		// For anything else, it must be a string.
		if err := unreadByte(r, ch); err != nil {
			return nil, err
		}
		return unmarshalString(r) // Call without passing `ch` here
	}
}

// unmarshalInt reads an integer from the Bencode data.
func unmarshalInt(r io.Reader) (int, error) {
	var buf bytes.Buffer
	for {
		ch, err := readByte(r)
		if err != nil {
			return 0, err
		}
		if ch == 'e' {
			break
		}
		buf.WriteByte(ch)
	}

	value, err := strconv.Atoi(buf.String())
	if err != nil {
		return 0, fmt.Errorf("invalid integer value: %v", err)
	}
	return value, nil
}

// unmarshalString reads a string from the Bencode data.
func unmarshalString(r io.Reader) (string, error) {
	lengthStr, err := readUntilColon(r)
	if err != nil {
		return "", err
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("invalid string length: %v", err)
	}

	strData := make([]byte, length)
	if _, err := io.ReadFull(r, strData); err != nil {
		return "", err
	}

	return string(strData), nil
}

// unmarshalList reads a list from the Bencode data.
func unmarshalList(r io.Reader) ([]any, error) {
	var list []any
	for {
		ch, err := readByte(r)
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			break
		}
		// Rewind the byte to read it correctly
		if err := unreadByte(r, ch); err != nil {
			return nil, err
		}
		value, err := unmarshalValue(r)
		if err != nil {
			return nil, err
		}
		list = append(list, value)
	}
	return list, nil
}

// unmarshalDict reads a dictionary from the Bencode data.
func unmarshalDict(r io.Reader) (map[string]any, error) {
	dict := make(map[string]any)
	for {
		ch, err := readByte(r)
		if err != nil {
			return nil, err
		}
		if ch == 'e' {
			break
		}
		// Rewind the byte to read it correctly
		if err := unreadByte(r, ch); err != nil {
			return nil, err
		}
		key, err := unmarshalString(r)
		if err != nil {
			return nil, err
		}
		value, err := unmarshalValue(r)
		if err != nil {
			return nil, err
		}
		dict[key] = value
	}
	return dict, nil
}

// Helper functions

// readByte reads a single byte from the reader.
func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := r.Read(b[:])
	return b[0], err
}

// unreadByte is a simple version that rewinds the read operation.
func unreadByte(r io.Reader, b byte) error {
	if seeker, ok := r.(io.Seeker); ok {
		_, err := seeker.Seek(-1, io.SeekCurrent)
		return err
	}
	return fmt.Errorf("unreadByte not supported for this reader")
}

// readUntilColon reads bytes until it encounters a colon.
func readUntilColon(r io.Reader) (string, error) {
	var buf bytes.Buffer
	for {
		ch, err := readByte(r)
		if err != nil {
			return "", err
		}
		if ch == ':' {
			break
		}
		buf.WriteByte(ch)
	}
	return buf.String(), nil
}
