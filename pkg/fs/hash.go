package fs

import (
	"crypto/sha1"
	"encoding/hex"
	"hash/crc32"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/xxh3"
)

func Uint64ToString(s uint64) string {
	var result []byte
	result = append(
		result,
		byte(s>>56),
		byte(s>>48),
		byte(s>>40),
		byte(s>>32),
		byte(s>>24),
		byte(s>>16),
		byte(s>>8),
		byte(s),
	)

	return hex.EncodeToString(result)

}

func XXHash3_Str(fileName string) string {
	s := XXHash3(fileName)
	if s == 0 {
		return ""
	}
	return Uint64ToString(s)
}
func XXHash3(fileName string) uint64 {
	//	var result []byte

	file, err := os.Open(fileName)

	if err != nil {
		//		return ""
		return 0
	}

	defer file.Close()

	hash := xxh3.New()

	if _, err := io.Copy(hash, file); err != nil {
		//		return ""
		return 0
	}

	s := hash.Sum64()
	return s
	/*	result = append(
			result,
			byte(s>>56),
			byte(s>>48),
			byte(s>>40),
			byte(s>>32),
			byte(s>>24),
			byte(s>>16),
			byte(s>>8),
			byte(s),
		)

		return hex.EncodeToString(result)
	*/
}
func HashXXH2_64(fileName string) string {
	var result []byte

	file, err := os.Open(fileName)

	if err != nil {
		return ""
	}

	defer file.Close()

	hash := xxhash.New()

	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(result))
}

// Hash returns the SHA1 hash of a file as string.
func Hash_(fileName string) string {
	var result []byte

	file, err := os.Open(fileName)

	if err != nil {
		return ""
	}

	defer file.Close()

	hash := sha1.New()

	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(result))
}

// Checksum returns the CRC32 checksum of a file as string.
func Checksum(fileName string) string {
	var result []byte

	file, err := os.Open(fileName)

	if err != nil {
		return ""
	}

	defer file.Close()

	hash := crc32.New(crc32.MakeTable(crc32.Castagnoli))

	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(result))
}

// IsHash tests if a string looks like a hash.
func IsHash(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if (r < 48 || r > 57) && (r < 97 || r > 102) && (r < 65 || r > 70) {
			return false
		}
	}

	switch len(s) {
	case 8, 16, 32, 40, 56, 64, 80, 128, 256:
		return true
	}

	return false
}
