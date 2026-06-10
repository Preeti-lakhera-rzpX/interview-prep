package wal

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"sync"
	"time"
)

// WAL provides write-ahead logging for crash recovery.
type WAL interface {
	Append(ctx context.Context, r Record) error
	Replay(ctx context.Context, fn func(Record) error) error
	Truncate(ctx context.Context) error
	Close() error
}

// FileWAL is a file-backed WAL implementation.
// Concurrency: a single mutex guards all writes. Replay is expected to run
// during startup before concurrent access begins.
type FileWAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// Open opens or creates a WAL file at the given path.
func Open(path string) (*FileWAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("wal open: %w", err)
	}
	return &FileWAL{file: f, path: path}, nil
}

// Append writes a record to the WAL.
// Binary format: [4B total_len][1B op][2B key_len][key][4B val_len][val][8B expiry_ns][4B crc32]
func (w *FileWAL) Append(_ context.Context, r Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	payload := encodeRecord(r)
	totalLen := uint32(len(payload) + 4) // +4 for the CRC at the end
	checksum := crc32.ChecksumIEEE(payload)

	buf := make([]byte, 4+len(payload)+4)
	binary.LittleEndian.PutUint32(buf[0:4], totalLen)
	copy(buf[4:4+len(payload)], payload)
	binary.LittleEndian.PutUint32(buf[4+len(payload):], checksum)

	if _, err := w.file.Write(buf); err != nil {
		return fmt.Errorf("wal append: %w", err)
	}
	return w.file.Sync()
}

// Replay reads all valid records and calls fn for each.
func (w *FileWAL) Replay(_ context.Context, fn func(Record) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, 0); err != nil {
		return fmt.Errorf("wal seek: %w", err)
	}

	for {
		var totalLen uint32
		if err := binary.Read(w.file, binary.LittleEndian, &totalLen); err != nil {
			break // EOF or corrupt trailing bytes
		}

		buf := make([]byte, totalLen)
		if _, err := w.file.Read(buf); err != nil {
			break
		}

		payloadLen := len(buf) - 4
		if payloadLen < 0 {
			break
		}
		payload := buf[:payloadLen]
		storedCRC := binary.LittleEndian.Uint32(buf[payloadLen:])

		if crc32.ChecksumIEEE(payload) != storedCRC {
			break // corrupt record, stop replay
		}

		rec, err := decodeRecord(payload)
		if err != nil {
			break
		}

		if err := fn(rec); err != nil {
			return fmt.Errorf("wal replay callback: %w", err)
		}
	}
	return nil
}

// Truncate resets the WAL file.
func (w *FileWAL) Truncate(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("wal truncate: %w", err)
	}
	_, err := w.file.Seek(0, 0)
	return err
}

// Close syncs and closes the WAL file.
func (w *FileWAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.file.Sync(); err != nil {
		return err
	}
	return w.file.Close()
}

func encodeRecord(r Record) []byte {
	keyLen := len(r.Key)
	valLen := len(r.Value)
	// 1B op + 2B keyLen + key + 4B valLen + val + 8B expiry
	buf := make([]byte, 1+2+keyLen+4+valLen+8)
	offset := 0

	buf[offset] = byte(r.Op)
	offset++

	binary.LittleEndian.PutUint16(buf[offset:], uint16(keyLen))
	offset += 2

	copy(buf[offset:], r.Key)
	offset += keyLen

	binary.LittleEndian.PutUint32(buf[offset:], uint32(valLen))
	offset += 4

	copy(buf[offset:], r.Value)
	offset += valLen

	binary.LittleEndian.PutUint64(buf[offset:], uint64(r.ExpiresAt.UnixNano()))
	return buf
}

func decodeRecord(buf []byte) (Record, error) {
	if len(buf) < 1+2 {
		return Record{}, fmt.Errorf("record too short")
	}
	var r Record
	offset := 0

	r.Op = OpType(buf[offset])
	offset++

	keyLen := int(binary.LittleEndian.Uint16(buf[offset:]))
	offset += 2

	if offset+keyLen > len(buf) {
		return Record{}, fmt.Errorf("key length exceeds record")
	}
	r.Key = string(buf[offset : offset+keyLen])
	offset += keyLen

	if offset+4 > len(buf) {
		return Record{}, fmt.Errorf("missing value length")
	}
	valLen := int(binary.LittleEndian.Uint32(buf[offset:]))
	offset += 4

	if offset+valLen > len(buf) {
		return Record{}, fmt.Errorf("value length exceeds record")
	}
	r.Value = make([]byte, valLen)
	copy(r.Value, buf[offset:offset+valLen])
	offset += valLen

	if offset+8 > len(buf) {
		return Record{}, fmt.Errorf("missing expiry")
	}
	nsec := int64(binary.LittleEndian.Uint64(buf[offset:]))
	if nsec > 0 {
		r.ExpiresAt = nsecToTime(nsec)
	}
	return r, nil
}

func nsecToTime(nsec int64) time.Time {
	sec := nsec / 1e9
	nano := nsec % 1e9
	return time.Unix(sec, nano)
}
