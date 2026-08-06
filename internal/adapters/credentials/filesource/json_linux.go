//go:build linux

package filesource

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"orquesta/internal/credentials"
)

// validateJSONObject checks syntax and duplicate decoded field names without
// decoding scalar values into immutable Go strings that could retain secrets.
func validateJSONObject(content []byte) error {
	if !utf8.Valid(content) || !json.Valid(content) {
		return credentials.NewError(credentials.ErrorUnsafeFile, "material_source_json")
	}
	parser := jsonObjectParser{content: content}
	parser.skipWhitespace()
	count, err := parser.parseObject(1)
	parser.skipWhitespace()
	if err != nil || count == 0 || parser.offset != len(content) {
		return credentials.NewError(credentials.ErrorUnsafeFile, "material_source_json")
	}
	return nil
}

type jsonObjectParser struct {
	content []byte
	offset  int
}

func (parser *jsonObjectParser) parseObject(depth int) (int, error) {
	if depth > 128 {
		return 0, errors.New("json nesting")
	}
	if !parser.take('{') {
		return 0, errors.New("json object open")
	}
	keys := make(map[[sha256.Size]byte][][]byte, 8)
	defer func() {
		for _, bucket := range keys {
			for _, key := range bucket {
				clear(key)
			}
		}
	}()
	count := 0
	parser.skipWhitespace()
	if parser.take('}') {
		return 0, nil
	}
	for {
		key, err := parser.readObjectKey()
		if err != nil {
			return 0, err
		}
		digest := sha256.Sum256(key)
		for _, prior := range keys[digest] {
			if bytes.Equal(prior, key) {
				clear(key)
				return 0, errors.New("json duplicate name")
			}
		}
		keys[digest] = append(keys[digest], key)
		parser.skipWhitespace()
		if !parser.take(':') {
			return 0, errors.New("json object separator")
		}
		if err := parser.parseValue(depth + 1); err != nil {
			return 0, err
		}
		count++
		parser.skipWhitespace()
		if parser.take('}') {
			return count, nil
		}
		if !parser.take(',') {
			return 0, errors.New("json object delimiter")
		}
		parser.skipWhitespace()
	}
}

func (parser *jsonObjectParser) parseValue(depth int) error {
	if depth > 128 {
		return errors.New("json nesting")
	}
	parser.skipWhitespace()
	if parser.offset >= len(parser.content) {
		return errors.New("json value")
	}
	switch parser.content[parser.offset] {
	case '{':
		_, err := parser.parseObject(depth)
		return err
	case '[':
		return parser.parseArray(depth)
	case '"':
		return parser.skipString()
	default:
		start := parser.offset
		for parser.offset < len(parser.content) {
			switch parser.content[parser.offset] {
			case ' ', '\t', '\r', '\n', ',', ']', '}':
				if parser.offset == start {
					return errors.New("json scalar")
				}
				return nil
			default:
				parser.offset++
			}
		}
		if parser.offset == start {
			return errors.New("json scalar")
		}
		return nil
	}
}

func (parser *jsonObjectParser) parseArray(depth int) error {
	if !parser.take('[') {
		return errors.New("json array open")
	}
	parser.skipWhitespace()
	if parser.take(']') {
		return nil
	}
	for {
		if err := parser.parseValue(depth + 1); err != nil {
			return err
		}
		parser.skipWhitespace()
		if parser.take(']') {
			return nil
		}
		if !parser.take(',') {
			return errors.New("json array delimiter")
		}
		parser.skipWhitespace()
	}
}

func (parser *jsonObjectParser) readObjectKey() ([]byte, error) {
	if !parser.take('"') {
		return nil, errors.New("json object name")
	}
	rawEnd, found := parser.stringEnd(parser.offset)
	if !found {
		return nil, errors.New("json object name")
	}
	key := make([]byte, 0, rawEnd-parser.offset)
	for parser.offset < len(parser.content) {
		value := parser.content[parser.offset]
		parser.offset++
		switch value {
		case '"':
			return key, nil
		case '\\':
			decoded, err := parser.readEscape()
			if err != nil {
				clear(key)
				return nil, err
			}
			key = append(key, decoded...)
			clear(decoded)
		default:
			key = append(key, value)
		}
	}
	clear(key)
	return nil, errors.New("json object name")
}

func (parser *jsonObjectParser) stringEnd(start int) (int, bool) {
	escaped := false
	for index := start; index < len(parser.content); index++ {
		if escaped {
			escaped = false
			continue
		}
		switch parser.content[index] {
		case '\\':
			escaped = true
		case '"':
			return index, true
		}
	}
	return 0, false
}

func (parser *jsonObjectParser) readEscape() ([]byte, error) {
	if parser.offset >= len(parser.content) {
		return nil, errors.New("json escape")
	}
	escaped := parser.content[parser.offset]
	parser.offset++
	switch escaped {
	case '"', '\\', '/':
		return []byte{escaped}, nil
	case 'b':
		return []byte{'\b'}, nil
	case 'f':
		return []byte{'\f'}, nil
	case 'n':
		return []byte{'\n'}, nil
	case 'r':
		return []byte{'\r'}, nil
	case 't':
		return []byte{'\t'}, nil
	case 'u':
		value, valid := parser.readHexRune()
		if !valid {
			return nil, errors.New("json unicode escape")
		}
		if value >= 0xd800 && value <= 0xdbff {
			if parser.offset+2 > len(parser.content) || parser.content[parser.offset] != '\\' || parser.content[parser.offset+1] != 'u' {
				return nil, errors.New("json unicode surrogate")
			}
			parser.offset += 2
			low, valid := parser.readHexRune()
			if !valid || low < 0xdc00 || low > 0xdfff {
				return nil, errors.New("json unicode surrogate")
			}
			value = 0x10000 + (value-0xd800)*0x400 + (low - 0xdc00)
		} else if value >= 0xdc00 && value <= 0xdfff {
			return nil, errors.New("json unicode surrogate")
		}
		var encoded [utf8.UTFMax]byte
		length := utf8.EncodeRune(encoded[:], rune(value))
		result := append([]byte(nil), encoded[:length]...)
		clear(encoded[:])
		return result, nil
	default:
		return nil, errors.New("json escape")
	}
}

func (parser *jsonObjectParser) readHexRune() (uint32, bool) {
	if parser.offset+4 > len(parser.content) {
		return 0, false
	}
	var value uint32
	for range 4 {
		value <<= 4
		current := parser.content[parser.offset]
		parser.offset++
		switch {
		case current >= '0' && current <= '9':
			value |= uint32(current - '0')
		case current >= 'a' && current <= 'f':
			value |= uint32(current-'a') + 10
		case current >= 'A' && current <= 'F':
			value |= uint32(current-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func (parser *jsonObjectParser) skipString() error {
	if !parser.take('"') {
		return errors.New("json string")
	}
	for parser.offset < len(parser.content) {
		value := parser.content[parser.offset]
		parser.offset++
		if value == '"' {
			return nil
		}
		if value == '\\' {
			if parser.offset >= len(parser.content) {
				return errors.New("json string escape")
			}
			escaped := parser.content[parser.offset]
			parser.offset++
			if escaped == 'u' {
				if parser.offset+4 > len(parser.content) {
					return errors.New("json string escape")
				}
				parser.offset += 4
			}
		}
	}
	return errors.New("json string")
}

func (parser *jsonObjectParser) skipWhitespace() {
	for parser.offset < len(parser.content) {
		switch parser.content[parser.offset] {
		case ' ', '\t', '\r', '\n':
			parser.offset++
		default:
			return
		}
	}
}

func (parser *jsonObjectParser) take(expected byte) bool {
	if parser.offset >= len(parser.content) || parser.content[parser.offset] != expected {
		return false
	}
	parser.offset++
	return true
}
