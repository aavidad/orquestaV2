// Este fichero traduce JSONL neutral a elementos revisables en memoria.
// Solo lee el inventario; no decide equivalencia ni escribe propuestas.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	maxInventoryFileBytes = 32 << 20
	maxInventoryLineBytes = 4 << 20
	maxInventoryItems     = 10_000
)

type inventory struct {
	path       string
	fileDigest string
	items      []inventoryItem
	byID       map[string]inventoryItem
	families   []string
}

type inventoryItem struct {
	ID            string
	Revision      string
	Title         string
	Summary       string
	Family        string
	Sources       []string
	Attempts      []string
	Uncertainties []string
	Raw           string
}

func loadInventory(filePath string) (*inventory, error) {
	content, err := readPrivateInventoryFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("leer inventario: %w", err)
	}
	result := &inventory{
		path:       filePath,
		fileDigest: digestBytes(content),
		byID:       map[string]inventoryItem{},
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), maxInventoryLineBytes)
	line := 0
	families := map[string]struct{}{}
	for scanner.Scan() {
		line++
		raw := bytes.TrimSpace(scanner.Bytes())
		if len(raw) == 0 {
			continue
		}
		if len(result.items) >= maxInventoryItems {
			return nil, fmt.Errorf("el inventario supera %d elementos semánticos", maxInventoryItems)
		}
		item, err := decodeInventoryItem(raw)
		if err != nil {
			return nil, fmt.Errorf("inventario línea %d: %w", line, err)
		}
		if _, exists := result.byID[item.ID]; exists {
			return nil, fmt.Errorf("identificador de inventario duplicado: %s", item.ID)
		}
		result.items = append(result.items, item)
		result.byID[item.ID] = item
		families[item.Family] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("recorrer inventario: %w", err)
	}
	sort.Slice(result.items, func(i, j int) bool {
		return result.items[i].ID < result.items[j].ID
	})
	for family := range families {
		result.families = append(result.families, family)
	}
	sort.Strings(result.families)
	return result, nil
}

func decodeInventoryItem(raw []byte) (inventoryItem, error) {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return inventoryItem{}, errors.New("JSON no válido")
	}
	revision := "sha256:" + digestBytes(raw)
	id := firstString(value, "id", "item_ref", "behavior_ref")
	if id == "" {
		return inventoryItem{}, errors.New("falta un identificador semántico estable")
	}
	title := firstString(value, "title", "name", "path", "summary")
	if title == "" {
		title = id
	}
	summary := firstString(value, "summary", "conducta_observada", "description")
	family := firstString(value, "family", "category", "type", "record_kind")
	if family == "" {
		family = "sin_clasificar"
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return inventoryItem{}, errors.New("registro no representable")
	}
	return inventoryItem{
		ID:       id,
		Revision: revision,
		Title:    title,
		Summary:  summary,
		Family:   family,
		Sources: collectValues(
			value,
			[]string{"sources", "fuentes"},
			[]string{
				"ref_name", "commit_id", "tree_id", "path", "canonical_source",
				"source_ref", "variant_ref", "occurrence_ref", "blob_id", "object_id",
			},
		),
		Attempts:      collectValues(value, []string{"attempts", "historical_attempts", "intentos"}, nil),
		Uncertainties: collectValues(value, []string{"uncertainties", "incertidumbres"}, nil),
		Raw:           string(pretty),
	}, nil
}

func (inventory *inventory) verifyUnchanged() error {
	content, err := readPrivateInventoryFile(inventory.path)
	if err != nil {
		return errInventoryChanged
	}
	if digestBytes(content) != inventory.fileDigest {
		return errInventoryChanged
	}
	return nil
}

func readPrivateInventoryFile(filePath string) ([]byte, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("inventario local inseguro")
	}
	if info.Size() > maxInventoryFileBytes {
		return nil, fmt.Errorf("inventario físico demasiado grande: máximo %d bytes", maxInventoryFileBytes)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return nil, errors.New("inventario local cambiante")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxInventoryFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxInventoryFileBytes {
		return nil, fmt.Errorf("inventario físico demasiado grande: máximo %d bytes", maxInventoryFileBytes)
	}
	finalInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, finalInfo) || finalInfo.Size() != int64(len(content)) {
		return nil, errors.New("inventario local cambiante")
	}
	return content, nil
}

func (inventory *inventory) find(id string) (inventoryItem, bool) {
	item, found := inventory.byID[id]
	return item, found
}

func firstString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func collectValues(value map[string]any, lists, scalars []string) []string {
	seen := map[string]struct{}{}
	var result []string
	var add func(any)
	add = func(candidate any) {
		switch typed := candidate.(type) {
		case string:
			text := strings.TrimSpace(typed)
			if text != "" {
				if _, exists := seen[text]; !exists {
					seen[text] = struct{}{}
					result = append(result, text)
				}
			}
		case []any:
			for _, member := range typed {
				add(member)
			}
		default:
			if typed != nil {
				content, err := json.Marshal(typed)
				if err == nil {
					add(string(content))
				}
			}
		}
	}
	for _, key := range lists {
		add(value[key])
	}
	for _, key := range scalars {
		add(value[key])
	}
	sort.Strings(result)
	return result
}

func digestBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
