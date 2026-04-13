package microprogramacionapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func validarArchivosMaterializables(item *EspecificacionFuncion, archivos []ArchivoEntrega) error {
	if item == nil {
		return fmt.Errorf("especificacion obligatoria")
	}
	for _, archivo := range archivos {
		ruta := strings.TrimSpace(archivo.RutaRelativa)
		if filepath.Ext(ruta) != ".go" {
			continue
		}
		if err := validarSintaxisGo(ruta, archivo.Contenido); err != nil {
			return err
		}
		if ruta == strings.TrimSpace(item.ArchivoObjetivo) {
			if err := validarSimboloObjetivoGo(item, archivo.Contenido); err != nil {
				return err
			}
		}
	}
	return nil
}

func validarSintaxisGo(ruta, contenido string) error {
	if strings.TrimSpace(contenido) == "" {
		return fmt.Errorf("contenido vacio para %q", ruta)
	}
	fs := token.NewFileSet()
	if _, err := parser.ParseFile(fs, ruta, contenido, parser.AllErrors); err != nil {
		return fmt.Errorf("contenido go invalido para %q: %w", ruta, err)
	}
	return nil
}

func validarSimboloObjetivoGo(item *EspecificacionFuncion, contenido string) error {
	simbolo := strings.TrimSpace(item.SimboloObjetivo)
	if simbolo == "" {
		return nil
	}
	fs := token.NewFileSet()
	archivo, err := parser.ParseFile(fs, strings.TrimSpace(item.ArchivoObjetivo), contenido, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("contenido go invalido para %q: %w", strings.TrimSpace(item.ArchivoObjetivo), err)
	}
	if contieneSimboloDeclaradoGo(archivo, simbolo) {
		return nil
	}
	return fmt.Errorf("el archivo objetivo %q no declara el simbolo %q", strings.TrimSpace(item.ArchivoObjetivo), simbolo)
}

func contieneSimboloDeclaradoGo(archivo *ast.File, simbolo string) bool {
	if archivo == nil || strings.TrimSpace(simbolo) == "" {
		return false
	}
	for _, decl := range archivo.Decls {
		switch actual := decl.(type) {
		case *ast.FuncDecl:
			if actual.Name != nil && actual.Name.Name == simbolo {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range actual.Specs {
				switch declarado := spec.(type) {
				case *ast.TypeSpec:
					if declarado.Name != nil && declarado.Name.Name == simbolo {
						return true
					}
				case *ast.ValueSpec:
					for _, nombre := range declarado.Names {
						if nombre != nil && nombre.Name == simbolo {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
