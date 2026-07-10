package auth

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DecodeAuthCode decodes the copy-paste authorization code the FABRIC
// credential manager displays when its browser page cannot reach the local CLI
// callback (for example on a remote or headless machine). The code is
// base64url-encoded, zlib-compressed token JSON in the TokenFile shape.
func DecodeAuthCode(code string) (TokenFile, error) {
	compact := stripWhitespace(code)
	if compact == "" {
		return TokenFile{}, fmt.Errorf("auth code: empty input")
	}
	raw, err := decodeBase64(compact)
	if err != nil {
		return TokenFile{}, fmt.Errorf("auth code: not valid base64: %w", err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return TokenFile{}, fmt.Errorf("auth code: not zlib-compressed: %w", err)
	}
	defer func() { _ = zr.Close() }()
	body, err := io.ReadAll(zr)
	if err != nil {
		return TokenFile{}, fmt.Errorf("auth code: decompressing: %w", err)
	}
	var tf TokenFile
	if err := json.Unmarshal(body, &tf); err != nil {
		return TokenFile{}, fmt.Errorf("auth code: parsing token JSON: %w", err)
	}
	if tf.IDToken == "" {
		return TokenFile{}, fmt.Errorf("auth code: decoded JSON has no id_token")
	}
	return tf, nil
}

// stripWhitespace removes all whitespace, so codes copied with line wraps or
// surrounding quotes-free padding still decode.
func stripWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
}

// decodeBase64 accepts url-safe or standard alphabets, padded or not.
func decodeBase64(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	} {
		if raw, err := enc.DecodeString(s); err == nil {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("no base64 alphabet matched")
}
