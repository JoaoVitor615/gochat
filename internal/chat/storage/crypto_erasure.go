package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
)

// CryptographicErasureBufferSize defines the I/O buffer chunk size for in-place overwrite.
const CryptographicErasureBufferSize = 64 * 1024

// CryptographicOverwrite overwrites all bytes of the target file in-place with
// ciphertext encrypted using AES-256-CTR with a cryptographically secure, purely random
// ephemeral key and IV. The key and IV are zeroed in memory immediately after completion.
// It also issues a filesystem sync to flush written blocks to physical storage.
func CryptographicOverwrite(filePath string) error {
	if filePath == "" {
		return errors.New("cryptographic overwrite: empty file path")
	}

	f, err := os.OpenFile(filePath, os.O_RDWR, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open file for cryptographic overwrite: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	size := stat.Size()
	if size == 0 {
		return nil
	}

	// 1. Generate purely random 256-bit (32 bytes) key and 16-byte IV
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("generate random AES-256 key: %w", err)
	}
	defer zeroBytes(key)

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return fmt.Errorf("generate random IV: %w", err)
	}
	defer zeroBytes(iv)

	// 2. Initialize AES-256-CTR stream cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create AES-256 cipher: %w", err)
	}

	stream := cipher.NewCTR(block, iv)

	// 3. Overwrite file in-place with encrypted blocks
	buf := make([]byte, CryptographicErasureBufferSize)
	defer zeroBytes(buf)

	var offset int64 = 0
	for offset < size {
		toRead := int64(len(buf))
		if offset+toRead > size {
			toRead = size - offset
		}

		n, err := f.ReadAt(buf[:toRead], offset)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read block at %d: %w", offset, err)
		}
		if n == 0 {
			break
		}

		// Encrypt block in-place
		stream.XORKeyStream(buf[:n], buf[:n])

		// Write back encrypted ciphertext at exact offset
		if _, err := f.WriteAt(buf[:n], offset); err != nil {
			return fmt.Errorf("write encrypted block at %d: %w", offset, err)
		}

		offset += int64(n)
	}

	// 4. Force filesystem buffers flush to physical storage
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync overwritten file to disk: %w", err)
	}

	return nil
}

// CryptographicErasureFile overwrites the file in-place with AES-256 ciphertext
// using an ephemeral random key, flushes to disk, and deletes the file from the filesystem.
func CryptographicErasureFile(filePath string) error {
	if err := CryptographicOverwrite(filePath); err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete erased file: %w", err)
	}

	return nil
}

// CryptographicErasure closes the database, securely overwrites all bytes with
// AES-256 encrypted data using an ephemeral random key, flushes to disk, and deletes the file.
func (s *BboltStore) CryptographicErasure() error {
	if s == nil || s.db == nil {
		return errors.New("cryptographic erasure: database not open")
	}

	path := s.db.Path()
	if path == "" {
		return errors.New("cryptographic erasure: empty database path")
	}

	// Close database first to release file lock and memory mappings
	if err := s.Close(); err != nil {
		return fmt.Errorf("close database before cryptographic erasure: %w", err)
	}

	return CryptographicErasureFile(path)
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
