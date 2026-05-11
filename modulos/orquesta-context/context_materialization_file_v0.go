package orquestacontext

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileContextRefStoreV0 struct {
	root string
}

func NewFileContextRefStoreV0(root string) (*FileContextRefStoreV0, []ContextMaterializationIssueV0) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRootInvalidoV0, "root", "root requerido"),
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRootInvalidoV0, "root", "root invalido"),
		}
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRootInvalidoV0, "root", "root no disponible"),
		}
	}
	return &FileContextRefStoreV0{root: abs}, nil
}

func (store *FileContextRefStoreV0) ReadContextRefV0(
	sourceRef string,
	maxBytes int,
) (ContextRefContentV0, []ContextMaterializationIssueV0) {
	sourceRef = strings.TrimSpace(sourceRef)
	if maxBytes <= 0 || maxBytes > maxContextMaterializedEntryBytesV0 {
		return ContextRefContentV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationTamanoV0, "max_bytes", "max_bytes invalido"),
		}
	}
	path, issues := store.resolveContextPathV0(sourceRef)
	if len(issues) > 0 {
		return ContextRefContentV0{}, issues
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return ContextRefContentV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRefNoEncontradaV0, sourceRef, "ref no encontrada"),
		}
	}
	if err != nil {
		return ContextRefContentV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRefInvalidaV0, sourceRef, "ref no disponible"),
		}
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes+1)))
	if err != nil {
		return ContextRefContentV0{}, []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRefInvalidaV0, sourceRef, "lectura fallida"),
		}
	}
	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	return ContextRefContentV0{
		SourceRef: sourceRef,
		Content:   string(data),
		Bytes:     len(data),
		Truncated: truncated,
	}, nil
}

func (store *FileContextRefStoreV0) resolveContextPathV0(sourceRef string) (string, []ContextMaterializationIssueV0) {
	if store == nil || store.root == "" {
		return "", []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRootInvalidoV0, "root", "root requerido"),
		}
	}
	if !validFileContextSourceRefV0(sourceRef) {
		return "", []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRefInvalidaV0, sourceRef, "ref invalida"),
		}
	}
	candidate := filepath.Join(store.root, filepath.FromSlash(sourceRef))
	rel, err := filepath.Rel(store.root, candidate)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", []ContextMaterializationIssueV0{
			contextMaterializationIssueV0(ErrContextMaterializationRefInvalidaV0, sourceRef, "ref fuera de root"),
		}
	}
	return candidate, nil
}

func validFileContextSourceRefV0(sourceRef string) bool {
	if sourceRef == "" ||
		strings.HasPrefix(sourceRef, "/") ||
		strings.Contains(sourceRef, "\\") ||
		strings.Contains(sourceRef, "://") ||
		strings.Contains(sourceRef, "..") ||
		strings.Contains(strings.ToLower(sourceRef), "$home") ||
		strings.Contains(sourceRef, "~") {
		return false
	}
	return strings.HasPrefix(sourceRef, "modulos/") && isRelativeContractPathLikeContextV0(sourceRef)
}

func isRelativeContractPathLikeContextV0(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			continue
		case r == '/', r == '.', r == '_', r == '-', r == '+', r == '=', r == '@', r == ',':
			continue
		default:
			return false
		}
	}
	return true
}
