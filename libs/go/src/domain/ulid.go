package domain

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Crockford's Base32 alphabet for ULID encoding.
const ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var ulidDecodeMap [256]byte

func init() {
	for i := range ulidDecodeMap {
		ulidDecodeMap[i] = 0xFF
	}
	for i, c := range ulidAlphabet {
		ulidDecodeMap[c] = byte(i)
		ulidDecodeMap[c+32] = byte(i) // lowercase
	}
}

// ULID represents a Universally Unique Lexicographically Sortable Identifier.
type ULID struct {
	value string
	time  uint64
}

var (
	ulidMu       sync.Mutex
	ulidLastTime uint64
	ulidLastRand [10]byte
)

// NewULID generates a new ULID with the current timestamp.
func NewULID() ULID {
	return NewULIDWithTime(time.Now())
}

// NewULIDWithTime generates a new ULID with the specified timestamp.
func NewULIDWithTime(t time.Time) ULID {
	ms := uint64(t.UnixMilli())

	ulidMu.Lock()
	defer ulidMu.Unlock()

	if ms == ulidLastTime {
		// Increment random part for same millisecond
		for i := 9; i >= 0; i-- {
			ulidLastRand[i]++
			if ulidLastRand[i] != 0 {
				break
			}
		}
	} else {
		ulidLastTime = ms
		rand.Read(ulidLastRand[:])
	}

	return ULID{
		value: encodeULID(ms, ulidLastRand),
		time:  ms,
	}
}

// ParseULID parses a ULID string.
func ParseULID(value string) (ULID, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if len(normalized) != 26 {
		return ULID{}, fmt.Errorf("ULID must be 26 characters, got %d", len(normalized))
	}
	for _, c := range normalized {
		if ulidDecodeMap[c] == 0xFF {
			return ULID{}, fmt.Errorf("invalid ULID character: %c", c)
		}
	}
	ts := decodeULIDTime(normalized)
	return ULID{value: normalized, time: ts}, nil
}

// MustParseULID parses a ULID string, panicking on invalid input.
func MustParseULID(value string) ULID {
	ulid, err := ParseULID(value)
	if err != nil {
		panic(err)
	}
	return ulid
}

// String returns the ULID as a string.
func (u ULID) String() string {
	return u.value
}

// Time returns the timestamp encoded in the ULID.
func (u ULID) Time() time.Time {
	return time.UnixMilli(int64(u.time))
}

// IsZero returns true if the ULID is zero.
func (u ULID) IsZero() bool {
	return u.value == "" || u.value == "00000000000000000000000000"
}

// Equals checks if two ULIDs are equal.
func (u ULID) Equals(other ULID) bool {
	return u.value == other.value
}

// Compare compares two ULIDs lexicographically.
func (u ULID) Compare(other ULID) int {
	return strings.Compare(u.value, other.value)
}

// MarshalJSON implements json.Marshaler.
func (u ULID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *ULID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ulid, err := ParseULID(s)
	if err != nil {
		return err
	}
	*u = ulid
	return nil
}

func encodeULID(ms uint64, random [10]byte) string {
	var result [26]byte
	// Encode timestamp (10 chars)
	result[0] = ulidAlphabet[(ms>>45)&0x1F]
	result[1] = ulidAlphabet[(ms>>40)&0x1F]
	result[2] = ulidAlphabet[(ms>>35)&0x1F]
	result[3] = ulidAlphabet[(ms>>30)&0x1F]
	result[4] = ulidAlphabet[(ms>>25)&0x1F]
	result[5] = ulidAlphabet[(ms>>20)&0x1F]
	result[6] = ulidAlphabet[(ms>>15)&0x1F]
	result[7] = ulidAlphabet[(ms>>10)&0x1F]
	result[8] = ulidAlphabet[(ms>>5)&0x1F]
	result[9] = ulidAlphabet[ms&0x1F]
	// Encode random (16 chars)
	result[10] = ulidAlphabet[(random[0]>>3)&0x1F]
	result[11] = ulidAlphabet[((random[0]&0x07)<<2)|((random[1]>>6)&0x03)]
	result[12] = ulidAlphabet[(random[1]>>1)&0x1F]
	result[13] = ulidAlphabet[((random[1]&0x01)<<4)|((random[2]>>4)&0x0F)]
	result[14] = ulidAlphabet[((random[2]&0x0F)<<1)|((random[3]>>7)&0x01)]
	result[15] = ulidAlphabet[(random[3]>>2)&0x1F]
	result[16] = ulidAlphabet[((random[3]&0x03)<<3)|((random[4]>>5)&0x07)]
	result[17] = ulidAlphabet[random[4]&0x1F]
	result[18] = ulidAlphabet[(random[5]>>3)&0x1F]
	result[19] = ulidAlphabet[((random[5]&0x07)<<2)|((random[6]>>6)&0x03)]
	result[20] = ulidAlphabet[(random[6]>>1)&0x1F]
	result[21] = ulidAlphabet[((random[6]&0x01)<<4)|((random[7]>>4)&0x0F)]
	result[22] = ulidAlphabet[((random[7]&0x0F)<<1)|((random[8]>>7)&0x01)]
	result[23] = ulidAlphabet[(random[8]>>2)&0x1F]
	result[24] = ulidAlphabet[((random[8]&0x03)<<3)|((random[9]>>5)&0x07)]
	result[25] = ulidAlphabet[random[9]&0x1F]
	return string(result[:])
}

func decodeULIDTime(s string) uint64 {
	var ts uint64
	ts |= uint64(ulidDecodeMap[s[0]]) << 45
	ts |= uint64(ulidDecodeMap[s[1]]) << 40
	ts |= uint64(ulidDecodeMap[s[2]]) << 35
	ts |= uint64(ulidDecodeMap[s[3]]) << 30
	ts |= uint64(ulidDecodeMap[s[4]]) << 25
	ts |= uint64(ulidDecodeMap[s[5]]) << 20
	ts |= uint64(ulidDecodeMap[s[6]]) << 15
	ts |= uint64(ulidDecodeMap[s[7]]) << 10
	ts |= uint64(ulidDecodeMap[s[8]]) << 5
	ts |= uint64(ulidDecodeMap[s[9]])
	return ts
}

// Bytes returns the ULID as a 16-byte array.
func (u ULID) Bytes() [16]byte {
	var result [16]byte
	// Decode timestamp (first 6 bytes)
	binary.BigEndian.PutUint64(result[0:8], u.time<<16)
	// Decode random (last 10 bytes) - simplified
	for i := 10; i < 26; i++ {
		result[6+(i-10)/2] |= ulidDecodeMap[u.value[i]] << (4 * ((i + 1) % 2))
	}
	return result
}
