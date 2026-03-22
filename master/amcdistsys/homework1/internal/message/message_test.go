package message

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

// --- Requirement: message size = 1024 bytes ---

func TestBuildMessage_Size(t *testing.T) {
	msg := BuildMessage(0)
	if len(msg.Bytes()) != MessageSize {
		t.Errorf("expected %d bytes, got %d", MessageSize, len(msg.Bytes()))
	}
}

// --- Requirement: byte 0 = sending node index ---

func TestBuildMessage_SenderIndex(t *testing.T) {
	for _, idx := range []uint8{0, 1, 5, 127, 255} {
		msg := BuildMessage(idx)
		if msg.SenderIndex() != idx {
			t.Errorf("sender index: expected %d, got %d", idx, msg.SenderIndex())
		}
	}
}

// --- Requirement: byte 0 is the sender index in the raw buffer ---

func TestBuildMessage_Byte0IsSenderIndex(t *testing.T) {
	msg := BuildMessage(42)
	if msg.Bytes()[0] != 42 {
		t.Errorf("raw byte 0: expected 42, got %d", msg.Bytes()[0])
	}
}

// --- Requirement: bytes 1004-1023 = SHA-1 of bytes 0-1003 ---

func TestBuildMessage_SHA1Correct(t *testing.T) {
	msg := BuildMessage(3)
	raw := msg.Bytes()
	computed := sha1.Sum(raw[:1004])
	for i := 0; i < 20; i++ {
		if raw[1004+i] != computed[i] {
			t.Fatalf("SHA-1 mismatch at byte %d: expected %02x, got %02x", 1004+i, computed[i], raw[1004+i])
		}
	}
}

// --- Verify should return OK for a valid message ---

func TestBuildMessage_VerifyOK(t *testing.T) {
	msg := BuildMessage(0)
	sentHex, calcHex, ok := msg.Verify()
	if !ok {
		t.Errorf("expected Verify to return ok=true for a freshly built message")
	}
	if sentHex != calcHex {
		t.Errorf("SHA-1 hex mismatch: sent=%s calc=%s", sentHex, calcHex)
	}
}

// --- Verify should return FAIL when payload is corrupted ---

func TestBuildMessage_VerifyFAIL_CorruptedPayload(t *testing.T) {
	msg := BuildMessage(0)
	// Flip a byte in the payload (byte 1-1003 range)
	msg.raw[500] ^= 0xFF
	_, _, ok := msg.Verify()
	if ok {
		t.Error("expected Verify to return ok=false after payload corruption")
	}
}

// --- Verify should return FAIL when SHA-1 bytes are corrupted ---

func TestBuildMessage_VerifyFAIL_CorruptedSHA1(t *testing.T) {
	msg := BuildMessage(0)
	// Flip a byte in the SHA-1 region (bytes 1004-1023)
	msg.raw[1010] ^= 0xFF
	_, _, ok := msg.Verify()
	if ok {
		t.Error("expected Verify to return ok=false after SHA-1 corruption")
	}
}

// --- Requirement: SHA-1 hex display format ---

func TestBuildMessage_VerifyHexFormat(t *testing.T) {
	msg := BuildMessage(7)
	sentHex, calcHex, _ := msg.Verify()
	// SHA-1 = 20 bytes = 40 hex characters
	if len(sentHex) != 40 {
		t.Errorf("sentHex length: expected 40, got %d", len(sentHex))
	}
	if len(calcHex) != 40 {
		t.Errorf("calcHex length: expected 40, got %d", len(calcHex))
	}
	// Verify they are valid hex
	if _, err := hex.DecodeString(sentHex); err != nil {
		t.Errorf("sentHex is not valid hex: %v", err)
	}
	if _, err := hex.DecodeString(calcHex); err != nil {
		t.Errorf("calcHex is not valid hex: %v", err)
	}
}

// --- ParseMessage: valid 1024-byte buffer ---

func TestParseMessage_Valid(t *testing.T) {
	original := BuildMessage(10)
	parsed, err := ParseMessage(original.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.SenderIndex() != 10 {
		t.Errorf("sender index: expected 10, got %d", parsed.SenderIndex())
	}
	_, _, ok := parsed.Verify()
	if !ok {
		t.Error("parsed message should verify ok")
	}
}

// --- ParseMessage: wrong size should error ---

func TestParseMessage_WrongSize(t *testing.T) {
	cases := []int{0, 1, 512, 1023, 1025, 2048}
	for _, size := range cases {
		buf := make([]byte, size)
		_, err := ParseMessage(buf)
		if err == nil {
			t.Errorf("expected error for buffer size %d", size)
		}
	}
}

// --- Bytes 1-1003 should contain random data (not all zeros) ---

func TestBuildMessage_RandomPayload(t *testing.T) {
	msg := BuildMessage(0)
	raw := msg.Bytes()
	allZero := true
	for i := 1; i < 1004; i++ {
		if raw[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("payload bytes 1-1003 should contain random data, but all are zero")
	}
}

// --- Two messages should have different random payloads ---

func TestBuildMessage_Uniqueness(t *testing.T) {
	msg1 := BuildMessage(0)
	msg2 := BuildMessage(0)
	same := true
	for i := 1; i < 1004; i++ {
		if msg1.Bytes()[i] != msg2.Bytes()[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("two built messages should have different random payloads")
	}
}

// --- Round-trip: build → bytes → parse → verify ---

func TestBuildMessage_RoundTrip(t *testing.T) {
	for idx := uint8(0); idx < 5; idx++ {
		msg := BuildMessage(idx)
		parsed, err := ParseMessage(msg.Bytes())
		if err != nil {
			t.Fatalf("node %d: parse error: %v", idx, err)
		}
		if parsed.SenderIndex() != idx {
			t.Errorf("node %d: sender index mismatch after round-trip", idx)
		}
		_, _, ok := parsed.Verify()
		if !ok {
			t.Errorf("node %d: verify failed after round-trip", idx)
		}
	}
}
