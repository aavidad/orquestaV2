// Este fichero impone JSON estricto y calcula huellas sin autorreferencia.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"unicode/utf8"
)

var (
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	refPattern    = regexp.MustCompile(`^(view|fence|policy|attempt|identity|binding|absence)_[0-9a-f]{64}$`)
)

func decodePinned(raw []byte, expected string, target any) error {
	if rawSHA256(raw) != expected {
		return contractFailure("base_no_fijada")
	}
	if !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return contractFailure("base_invalida")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return contractFailure("base_invalida")
	}
	return nil
}

func decodeCanonicalCandidate(raw []byte) (mappingCandidate, error) {
	var value mappingCandidate
	if !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) {
		return value, contractFailure("json_invalido")
	}
	if err := validateJSONBounds(raw); err != nil {
		return value, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, contractFailure("json_invalido")
	}
	if err := requireJSONEnd(decoder); err != nil {
		return value, err
	}
	canonical, err := canonicalJSON(value)
	if err != nil || !bytes.Equal(canonical, raw) {
		return value, contractFailure("json_no_canonico")
	}
	return value, nil
}

func validateJSONBounds(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	tokens := 0
	if err := scanJSONValue(decoder, 1, &tokens); err != nil {
		return contractFailure("json_invalido")
	}
	return requireJSONEnd(decoder)
}

func scanJSONValue(decoder *json.Decoder, depth int, tokens *int) error {
	if depth > maxJSONDepth || *tokens >= maxJSONTokens {
		return errors.New("límite JSON excedido")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	*tokens++
	if text, ok := token.(string); ok && len(text) > maxStringBytes {
		return errors.New("texto JSON demasiado largo")
	}
	delim, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delim == '[' {
		return scanJSONArray(decoder, depth, tokens)
	}
	if delim != '{' {
		return errors.New("delimitador JSON inesperado")
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		keyToken, keyErr := decoder.Token()
		key, valid := keyToken.(string)
		if keyErr != nil || !valid || len(key) > maxStringBytes || len(seen) >= maxArrayItems {
			return errors.New("clave JSON inválida")
		}
		*tokens++
		if _, duplicate := seen[key]; duplicate {
			return errors.New("clave JSON duplicada")
		}
		seen[key] = struct{}{}
		if err := scanJSONValue(decoder, depth+1, tokens); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func scanJSONArray(decoder *json.Decoder, depth int, tokens *int) error {
	count := 0
	for decoder.More() {
		count++
		if count > maxArrayItems {
			return errors.New("matriz JSON demasiado larga")
		}
		if err := scanJSONValue(decoder, depth+1, tokens); err != nil {
			return err
		}
	}
	_, err := decoder.Token()
	return err
}

func requireJSONEnd(decoder *json.Decoder) error {
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return contractFailure("json_invalido")
	}
	return nil
}

func canonicalJSON(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func rawSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func domainDigest(domain string, value any) (string, error) {
	raw, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(raw)
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func opaqueRef(value, prefix string) bool {
	return refPattern.MatchString(value) && len(value) > len(prefix) &&
		value[:len(prefix)+1] == prefix+"_"
}
