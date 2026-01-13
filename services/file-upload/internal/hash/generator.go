package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Generator generates SHA256 hashes for files
type Generator struct{}

// NewGenerator creates a new hash generator
func NewGenerator() *Generator {
	return &Generator{}
}

// ComputeHash computes SHA256 hash from reader (streaming)
func (g *Generator) ComputeHash(content io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, content); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ComputeHashWithSize computes hash and returns size
func (g *Generator) ComputeHashWithSize(content io.Reader) (hash string, size int64, err error) {
	hasher := sha256.New()
	size, err = io.Copy(hasher, content)
	if err != nil {
		return "", 0, err
	}
	hash = hex.EncodeToString(hasher.Sum(nil))
	return hash, size, nil
}

// ComputeHashFromBytes computes hash from byte slice
func (g *Generator) ComputeHashFromBytes(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyHash verifies content matches expected hash
func (g *Generator) VerifyHash(content io.Reader, expectedHash string) (bool, error) {
	computedHash, err := g.ComputeHash(content)
	if err != nil {
		return false, err
	}
	return computedHash == expectedHash, nil
}

// VerifyHashFromBytes verifies byte slice matches expected hash
func (g *Generator) VerifyHashFromBytes(data []byte, expectedHash string) bool {
	computedHash := g.ComputeHashFromBytes(data)
	return computedHash == expectedHash
}

// HashReader wraps a reader to compute hash while reading
type HashReader struct {
	reader io.Reader
	hasher *sha256.Digest
	size   int64
}

// NewHashReader creates a reader that computes hash while reading
func NewHashReader(r io.Reader) *HashReader {
	h := sha256.New()
	return &HashReader{
		reader: r,
		hasher: h.(*sha256.Digest),
		size:   0,
	}
}

// Read implements io.Reader
func (hr *HashReader) Read(p []byte) (n int, err error) {
	n, err = hr.reader.Read(p)
	if n > 0 {
		hr.hasher.Write(p[:n])
		hr.size += int64(n)
	}
	return n, err
}

// Hash returns the computed hash after reading is complete
func (hr *HashReader) Hash() string {
	return hex.EncodeToString(hr.hasher.Sum(nil))
}

// Size returns the total bytes read
func (hr *HashReader) Size() int64 {
	return hr.size
}

// TeeHashReader wraps a reader to compute hash while also writing to another writer
type TeeHashReader struct {
	reader io.Reader
	writer io.Writer
	hasher *sha256.Digest
	size   int64
}

// NewTeeHashReader creates a reader that computes hash and tees to writer
func NewTeeHashReader(r io.Reader, w io.Writer) *TeeHashReader {
	h := sha256.New()
	return &TeeHashReader{
		reader: r,
		writer: w,
		hasher: h.(*sha256.Digest),
		size:   0,
	}
}

// Read implements io.Reader
func (thr *TeeHashReader) Read(p []byte) (n int, err error) {
	n, err = thr.reader.Read(p)
	if n > 0 {
		thr.hasher.Write(p[:n])
		if thr.writer != nil {
			thr.writer.Write(p[:n])
		}
		thr.size += int64(n)
	}
	return n, err
}

// Hash returns the computed hash
func (thr *TeeHashReader) Hash() string {
	return hex.EncodeToString(thr.hasher.Sum(nil))
}

// Size returns the total bytes read
func (thr *TeeHashReader) Size() int64 {
	return thr.size
}
