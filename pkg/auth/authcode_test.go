package auth

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func encodeAuthCode(t *testing.T, tf TokenFile, enc *base64.Encoding) string {
	t.Helper()
	body, err := json.Marshal(tf)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		t.Fatalf("zlib writer: %v", err)
	}
	if _, err := zw.Write(body); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return enc.EncodeToString(buf.Bytes())
}

func TestDecodeAuthCodeRoundTrip(t *testing.T) {
	t.Parallel()
	want := TokenFile{IDToken: "a.b.c", RefreshToken: "rt-1", State: "valid", ExpiresAt: "later"}
	for name, enc := range map[string]*base64.Encoding{
		"raw-url":  base64.RawURLEncoding,
		"url":      base64.URLEncoding,
		"raw-std":  base64.RawStdEncoding,
		"standard": base64.StdEncoding,
	} {
		code := encodeAuthCode(t, want, enc)
		got, err := DecodeAuthCode(code)
		if err != nil {
			t.Errorf("%s: DecodeAuthCode: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("%s: got %+v, want %+v", name, got, want)
		}
	}
}

func TestDecodeAuthCodeToleratesWrappedPaste(t *testing.T) {
	t.Parallel()
	want := TokenFile{IDToken: "a.b.c", RefreshToken: "rt"}
	code := encodeAuthCode(t, want, base64.RawURLEncoding)
	wrapped := "  " + code[:10] + "\n" + code[10:20] + "\r\n" + code[20:] + " \n"
	got, err := DecodeAuthCode(wrapped)
	if err != nil {
		t.Fatalf("DecodeAuthCode(wrapped): %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDecodeAuthCodeRejectsGarbage(t *testing.T) {
	t.Parallel()
	for name, code := range map[string]string{
		"empty":       "",
		"not-base64":  "!!!not/base64!!!",
		"not-zlib":    base64.RawURLEncoding.EncodeToString([]byte("plain text")),
		"no-id-token": encodeAuthCode(t, TokenFile{RefreshToken: "only"}, base64.RawURLEncoding),
	} {
		if _, err := DecodeAuthCode(code); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDecodeAuthCodeRejectsMissingIDTokenViaRawJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"unexpected":"shape"}`))
	_ = zw.Close()
	if _, err := DecodeAuthCode(base64.RawURLEncoding.EncodeToString(buf.Bytes())); err == nil {
		t.Fatal("expected error for JSON without id_token")
	}
}
