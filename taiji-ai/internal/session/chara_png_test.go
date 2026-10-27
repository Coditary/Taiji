package session

import (
	"encoding/base64"
	"testing"
)

func TestParsePNGCharaTextJSON(t *testing.T) {
	payload := []byte(`{"data":{"name":"Hero"}}`)
	chunk := append(append([]byte("chara"), 0), payload...)
	png := buildMinimalPNG("tEXt", chunk)
	got, err := parsePNGCharaText(png)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got=%q", got)
	}
}

func TestParsePNGCharaTextBase64(t *testing.T) {
	raw := []byte(`{"name":"x"}`)
	enc := base64.StdEncoding.EncodeToString(raw)
	chunk := append(append([]byte("chara"), 0), enc...)
	png := buildMinimalPNG("tEXt", chunk)
	got, err := parsePNGCharaText(png)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("got=%q", got)
	}
}

func TestSplitPNGText(t *testing.T) {
	key, val, ok := splitPNGText([]byte("chara\x00hello"))
	if !ok || key != "chara" || val != "hello" {
		t.Fatalf("key=%q val=%q ok=%v", key, val, ok)
	}
}

func buildMinimalPNG(chunkType string, data []byte) []byte {
	sig := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	length := len(data)
	chunk := make([]byte, 0, 8+length+4)
	chunk = append(chunk, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	chunk = append(chunk, chunkType[0], chunkType[1], chunkType[2], chunkType[3])
	chunk = append(chunk, data...)
	chunk = append(chunk, 0, 0, 0, 0) // CRC placeholder
	return append(sig, chunk...)
}
