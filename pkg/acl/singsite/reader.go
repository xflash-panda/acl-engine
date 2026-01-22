package singsite

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// ItemType represents the type of a geosite rule item.
type ItemType = uint8

const (
	// RuleTypeDomain matches exact domain.
	RuleTypeDomain ItemType = iota
	// RuleTypeDomainSuffix matches domain suffix.
	RuleTypeDomainSuffix
	// RuleTypeDomainKeyword matches domain keyword.
	RuleTypeDomainKeyword
	// RuleTypeDomainRegex matches domain by regex.
	RuleTypeDomainRegex
)

// Item represents a single geosite rule item.
type Item struct {
	Type  ItemType
	Value string
}

// Reader reads sing-geosite format database files.
type Reader struct {
	reader       io.ReadSeeker
	domainIndex  map[string]int
	domainLength map[string]int
}

// LoadFromBytes loads a sing-geosite database from bytes.
func LoadFromBytes(data []byte) (reader *Reader, codes []string, err error) {
	reader = &Reader{
		reader: bytes.NewReader(data),
	}
	codes, err = reader.loadMetadata()
	if err != nil {
		return nil, nil, err
	}
	return reader, codes, nil
}

// LoadFromFile loads a sing-geosite database from a file.
func LoadFromFile(path string) (*Reader, []string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	reader := &Reader{
		reader: file,
	}
	codes, err := reader.loadMetadata()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return reader, codes, nil
}

// Close closes the underlying reader if it implements io.Closer.
func (r *Reader) Close() error {
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (r *Reader) loadMetadata() ([]string, error) {
	version, err := readByte(r.reader)
	if err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if version != 0 {
		return nil, errors.New("unknown sing-geosite version")
	}

	entryLength, err := readUvarint(r.reader)
	if err != nil {
		return nil, fmt.Errorf("read entry length: %w", err)
	}

	codes := make([]string, 0, entryLength)
	domainIndex := make(map[string]int)
	domainLength := make(map[string]int)

	for i := uint64(0); i < entryLength; i++ {
		code, err := readVString(r.reader)
		if err != nil {
			return nil, fmt.Errorf("read code: %w", err)
		}

		codeIndex, err := readUvarint(r.reader)
		if err != nil {
			return nil, fmt.Errorf("read code index: %w", err)
		}

		codeLength, err := readUvarint(r.reader)
		if err != nil {
			return nil, fmt.Errorf("read code length: %w", err)
		}

		codes = append(codes, code)
		domainIndex[code] = int(codeIndex)   //nolint:gosec // codeIndex is file offset, safe for reasonable file sizes
		domainLength[code] = int(codeLength) //nolint:gosec // codeLength is item count, safe for reasonable file sizes
	}

	r.domainIndex = domainIndex
	r.domainLength = domainLength
	return codes, nil
}

// Read reads all items for the given code.
func (r *Reader) Read(code string) ([]Item, error) {
	index, exists := r.domainIndex[code]
	if !exists {
		return nil, fmt.Errorf("code %q not found", code)
	}

	_, err := r.reader.Seek(int64(index), io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("seek to index: %w", err)
	}

	counter := &readCounter{Reader: r.reader}
	items := make([]Item, r.domainLength[code])

	for i := range items {
		itemType, err := readByte(counter)
		if err != nil {
			return nil, fmt.Errorf("read item type: %w", err)
		}

		value, err := readVString(counter)
		if err != nil {
			return nil, fmt.Errorf("read item value: %w", err)
		}

		items[i] = Item{
			Type:  itemType,
			Value: value,
		}
	}

	// Seek back to original position
	_, err = r.reader.Seek(int64(-index)-counter.count, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("seek back: %w", err)
	}

	return items, nil
}

// readCounter wraps an io.Reader and counts bytes read.
type readCounter struct {
	io.Reader
	count int64
}

func (r *readCounter) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.count += int64(n)
	return
}

// readByte reads a single byte from the reader.
func readByte(reader io.Reader) (byte, error) {
	if br, ok := reader.(io.ByteReader); ok {
		return br.ReadByte()
	}
	var b [1]byte
	_, err := io.ReadFull(reader, b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// byteReader wraps an io.Reader to provide io.ByteReader interface.
type byteReader struct {
	io.Reader
}

func (r byteReader) ReadByte() (byte, error) {
	return readByte(r.Reader)
}

// readUvarint reads a varint-encoded uint64 from the reader.
func readUvarint(reader io.Reader) (uint64, error) {
	if br, ok := reader.(io.ByteReader); ok {
		return binary.ReadUvarint(br)
	}
	return binary.ReadUvarint(byteReader{reader})
}

// readVString reads a varint-length-prefixed string from the reader.
func readVString(reader io.Reader) (string, error) {
	length, err := readUvarint(reader)
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
