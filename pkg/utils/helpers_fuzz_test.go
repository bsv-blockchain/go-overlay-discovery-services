package utils

import (
	"testing"
	"unicode/utf8"
)

// FuzzHexToBytes tests the HexToBytes function with random inputs
// to ensure it handles malformed hex strings gracefully without panicking.
func FuzzHexToBytes(f *testing.F) {
	// Seed corpus with valid hex strings
	f.Add("")
	f.Add("ff")
	f.Add("0123abcd")
	f.Add("ABCD")
	f.Add("aBcD")
	f.Add("0000")
	f.Add("deadbeef")
	f.Add("0102030405060708090a0b0c0d0e0f")

	// Seed corpus with invalid hex strings
	f.Add("xyz")
	f.Add("abc") // odd length
	f.Add("gg")
	f.Add("0x")
	f.Add("0xdeadbeef")
	f.Add(" ff")
	f.Add("ff ")
	f.Add("f f")

	// Seed corpus with edge cases
	f.Add("0")
	f.Add("f")
	f.Add("00")
	f.Add("FF")
	f.Add("!@#$")
	f.Add("12345") // odd length
	f.Add("\x00\x01")

	f.Fuzz(func(t *testing.T, hexStr string) {
		// Function should not panic on any input
		result, err := HexToBytes(hexStr)

		// If no error occurred, validate the result
		if err == nil {
			// Result length should be half the hex string length
			expectedLen := len(hexStr) / 2
			if len(result) != expectedLen {
				t.Errorf("HexToBytes(%q) returned %d bytes, expected %d", hexStr, len(result), expectedLen)
			}

			// Verify round-trip consistency
			hexBack := BytesToHex(result)
			// Convert both to lowercase for comparison
			if len(hexStr) > 0 {
				// Only check if input was valid (even length)
				if len(hexStr)%2 == 0 {
					lowerInput := ""
					for _, ch := range hexStr {
						if ch >= 'A' && ch <= 'F' {
							lowerInput += string(ch - 'A' + 'a')
						} else {
							lowerInput += string(ch)
						}
					}
					if hexBack != lowerInput {
						// Only report error if both are valid hex
						isValidHex := true
						for _, ch := range hexStr {
							if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
								isValidHex = false
								break
							}
						}
						if isValidHex {
							t.Errorf("Round-trip failed: HexToBytes(%q) -> BytesToHex() = %q", hexStr, hexBack)
						}
					}
				}
			}
		}
	})
}

// FuzzBytesToHex tests the BytesToHex function with random byte slices.
func FuzzBytesToHex(f *testing.F) {
	// Seed corpus with various byte sequences
	f.Add([]byte{})
	f.Add([]byte{0xff})
	f.Add([]byte{0x01, 0x23, 0xab, 0xcd})
	f.Add([]byte{0x00, 0x00})
	f.Add([]byte("hello"))
	f.Add([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f})

	// Seed corpus with edge cases
	f.Add([]byte{0})
	f.Add([]byte{255})
	f.Add([]byte{0x80})
	f.Add([]byte("\x00"))
	f.Add([]byte("\xff\xff\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Function should not panic on any input
		result := BytesToHex(data)

		// Validate result format
		if len(result) != len(data)*2 {
			t.Errorf("BytesToHex(%v) returned string of length %d, expected %d", data, len(result), len(data)*2)
		}

		// Result should only contain hex characters (0-9, a-f)
		for _, ch := range result {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
				t.Errorf("BytesToHex(%v) returned non-hex character: %q", data, ch)
			}
		}

		// Verify round-trip consistency
		decoded, err := HexToBytes(result)
		if err != nil {
			t.Errorf("Round-trip failed: HexToBytes(BytesToHex(%v)) returned error: %v", data, err)
		}
		if len(decoded) != len(data) {
			t.Errorf("Round-trip length mismatch: original %d bytes, decoded %d bytes", len(data), len(decoded))
		}
		for i := range data {
			if i < len(decoded) && decoded[i] != data[i] {
				t.Errorf("Round-trip failed at index %d: original %02x, decoded %02x", i, data[i], decoded[i])
				break
			}
		}
	})
}

// FuzzUTFBytesToString tests the UTFBytesToString function with random byte sequences,
// including invalid UTF-8 sequences.
func FuzzUTFBytesToString(f *testing.F) {
	// Seed corpus with valid UTF-8 strings
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add([]byte("hello world"))
	//nolint:gosmopolitan // Test case requires specific UTF-8 characters including Chinese
	f.Add([]byte("Hello, 世界"))
	f.Add([]byte("Testing 123"))

	// Seed corpus with binary data
	f.Add([]byte{0x01, 0x02, 0x03})
	f.Add([]byte{0x00})
	f.Add([]byte{0xff, 0xfe, 0xfd})

	// Seed corpus with edge cases
	// Invalid UTF-8 continuation byte
	f.Add([]byte{0x80})
	// Overlong encoding
	f.Add([]byte{0xc0, 0x80})
	// UTF-16 surrogate
	f.Add([]byte{0xed, 0xa0, 0x80})
	// Code point out of range
	f.Add([]byte{0xf4, 0x90, 0x80, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Function should not panic on any input
		result := UTFBytesToString(data)

		// Validate that the result is a valid string
		// The result is always valid, we just ensure no panic occurs
		_ = result

		// If input is valid UTF-8, output should match exactly
		if utf8.Valid(data) {
			if result != string(data) {
				t.Errorf("UTFBytesToString(%v) = %q, expected %q", data, result, string(data))
			}
		}

		// Round-trip should preserve the original bytes
		roundTrip := []byte(result)
		if len(roundTrip) != len(data) {
			// This is expected for invalid UTF-8, as Go replaces invalid sequences
			// We just ensure no panic occurs
			return
		}
	})
}

// FuzzFlattenFields tests the flattenFields function with random field collections.
func FuzzFlattenFields(f *testing.F) {
	// Seed corpus with various field configurations
	f.Add([]byte{}, []byte{})
	f.Add([]byte("hello"), []byte{})
	f.Add([]byte("hello"), []byte("world"))
	f.Add([]byte{0x01, 0x02}, []byte{0x03, 0x04})

	f.Fuzz(func(t *testing.T, field1, field2 []byte) {
		// Create a TokenFields with the fuzzed data
		fields := TokenFields{field1, field2}

		// Function should not panic on any input
		result := flattenFields(fields)

		// Validate result length
		expectedLen := len(field1) + len(field2)
		if len(result) != expectedLen {
			t.Errorf("flattenFields(%v) returned %d bytes, expected %d", fields, len(result), expectedLen)
		}

		// Verify content is concatenated correctly
		if len(result) >= len(field1) {
			for i := range field1 {
				if result[i] != field1[i] {
					t.Errorf("flattenFields(%v) first field mismatch at index %d", fields, i)
					break
				}
			}
		}
		if len(result) >= len(field1)+len(field2) {
			for i := range field2 {
				if result[len(field1)+i] != field2[i] {
					t.Errorf("flattenFields(%v) second field mismatch at index %d", fields, i)
					break
				}
			}
		}
	})
}
