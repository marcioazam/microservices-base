package property

import (
	"bytes"
	"testing"

	"github.com/auth-platform/file-upload/internal/hash"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: file-upload-service, Property 5: Hash Computation Correctness
// Validates: Requirements 5.1
// For any uploaded file, the computed SHA256 hash SHALL be deterministic—
// uploading the same file content SHALL always produce the same hash value.

func TestHashComputationProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	g := hash.NewGenerator()

	// Property: Same content produces same hash (deterministic)
	properties.Property("same content produces same hash", prop.ForAll(
		func(data []byte) bool {
			hash1 := g.ComputeHashFromBytes(data)
			hash2 := g.ComputeHashFromBytes(data)
			return hash1 == hash2
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: Different content produces different hash (collision resistance)
	properties.Property("different content produces different hash", prop.ForAll(
		func(data1, data2 []byte) bool {
			// Skip if data is the same
			if bytes.Equal(data1, data2) {
				return true
			}

			hash1 := g.ComputeHashFromBytes(data1)
			hash2 := g.ComputeHashFromBytes(data2)
			return hash1 != hash2
		},
		gen.SliceOfN(32, gen.UInt8()),
		gen.SliceOfN(32, gen.UInt8()),
	))

	// Property: Hash length is always 64 characters (SHA256 hex)
	properties.Property("hash length is 64 characters", prop.ForAll(
		func(data []byte) bool {
			h := g.ComputeHashFromBytes(data)
			return len(h) == 64
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: Hash contains only hex characters
	properties.Property("hash contains only hex characters", prop.ForAll(
		func(data []byte) bool {
			h := g.ComputeHashFromBytes(data)
			for _, c := range h {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: Streaming hash equals direct hash
	properties.Property("streaming hash equals direct hash", prop.ForAll(
		func(data []byte) bool {
			directHash := g.ComputeHashFromBytes(data)

			reader := bytes.NewReader(data)
			streamHash, err := g.ComputeHash(reader)
			if err != nil {
				return false
			}

			return directHash == streamHash
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: Hash verification works correctly
	properties.Property("hash verification works correctly", prop.ForAll(
		func(data []byte) bool {
			expectedHash := g.ComputeHashFromBytes(data)

			// Verify with correct hash
			reader := bytes.NewReader(data)
			valid, err := g.VerifyHash(reader, expectedHash)
			if err != nil || !valid {
				return false
			}

			// Verify with incorrect hash
			wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
			reader = bytes.NewReader(data)
			invalid, err := g.VerifyHash(reader, wrongHash)
			if err != nil {
				return false
			}

			// Should be invalid unless data happens to hash to all zeros (extremely unlikely)
			return !invalid || expectedHash == wrongHash
		},
		gen.SliceOfN(32, gen.UInt8()),
	))

	// Property: HashReader produces correct hash
	properties.Property("HashReader produces correct hash", prop.ForAll(
		func(data []byte) bool {
			expectedHash := g.ComputeHashFromBytes(data)

			reader := bytes.NewReader(data)
			hr := hash.NewHashReader(reader)

			// Read all data
			buf := make([]byte, len(data)+10)
			totalRead := 0
			for {
				n, err := hr.Read(buf[totalRead:])
				totalRead += n
				if err != nil {
					break
				}
			}

			return hr.Hash() == expectedHash && hr.Size() == int64(len(data))
		},
		gen.SliceOf(gen.UInt8()),
	))

	// Property: ComputeHashWithSize returns correct size
	properties.Property("ComputeHashWithSize returns correct size", prop.ForAll(
		func(data []byte) bool {
			reader := bytes.NewReader(data)
			h, size, err := g.ComputeHashWithSize(reader)
			if err != nil {
				return false
			}

			expectedHash := g.ComputeHashFromBytes(data)
			return h == expectedHash && size == int64(len(data))
		},
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

// TestHashDeterminism specifically tests determinism across multiple calls
func TestHashDeterminism(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	g := hash.NewGenerator()

	// Property: Multiple hash computations are identical
	properties.Property("multiple computations are identical", prop.ForAll(
		func(data []byte, iterations int) bool {
			if iterations < 2 {
				iterations = 2
			}
			if iterations > 10 {
				iterations = 10
			}

			firstHash := g.ComputeHashFromBytes(data)
			for i := 1; i < iterations; i++ {
				h := g.ComputeHashFromBytes(data)
				if h != firstHash {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.UInt8()),
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t)
}
