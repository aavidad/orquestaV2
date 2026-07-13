package orquestaruntimerequiredtest

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Este guard no cuenta texto. Sigue, mediante AST y go/types, la autoridad
// ejecutable (map[string]*allowedCommandIdentityV0) aunque se esconda tras
// selectores, alias, helpers, wrappers o una copia del mapa. Los mapas de
// configuracion map[string]string no son autoridad: no pueden producir la
// identidad inmutable que exige la ejecucion.
const propietarioAutoridadAllowlistV0 = "allowedCommandIdentityRegistryV0.resolveTokens"

const maxSitiosAutoridadAllowlistV0 = 1

type sitioAutoridadAllowlistV0 struct {
	Fichero     string
	Funcion     string
	Propietario string
	Linea       int
}

type analizadorAutoridadAllowlistV0 struct {
	fset       *token.FileSet
	files      []*ast.File
	names      map[*ast.File]string
	info       *types.Info
	tainted    map[types.Object]bool
	funcAlias  map[types.Object]*types.Func
	returnsMap map[*types.Func]bool
}

func TestLaAllowlistNoSeResuelveEnMasSitiosV0(t *testing.T) {
	ficheros, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	sources := make(map[string]string)
	for _, fichero := range ficheros {
		if strings.HasSuffix(fichero, "_test.go") {
			continue
		}
		contenido, err := os.ReadFile(fichero)
		if err != nil {
			t.Fatalf("leyendo %s: %v", fichero, err)
		}
		sources[fichero] = string(contenido)
	}
	sitios, err := analizarAutoridadAllowlistV0(sources)
	if err != nil {
		t.Fatal(err)
	}
	if err := validarPropietariosAutoridadAllowlistV0(sitios); err != nil {
		t.Fatal(err)
	}
}

func TestGuardAutoridadAllowlistV0DetectaSelectorAliasHelperWrapperYCopia(t *testing.T) {
	casos := map[string]string{
		"selector": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 { return r.commands[key] }`,
		"alias": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 { m := r.commands; return m[key] }`,
		"helper": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func lookup(m map[string]*allowedCommandIdentityV0, key string) *allowedCommandIdentityV0 { return m[key] }
func rogue(r registry, key string) *allowedCommandIdentityV0 { return lookup(r.commands, key) }`,
		"wrapper": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func lookup(m map[string]*allowedCommandIdentityV0, key string) *allowedCommandIdentityV0 { return m[key] }
func rogue(r registry, key string) *allowedCommandIdentityV0 { f := lookup; return f(r.commands, key) }`,
		"copia": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 {
	clone := map[string]*allowedCommandIdentityV0{}
	for k, v := range r.commands { clone[k] = v }
	return clone[key]
}`,
		"maps_clone": `package fixture
import "maps"
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 {
	clone := maps.Clone(r.commands)
	return clone[key]
}`,
		"conversion_mapa_nombrado": `package fixture
type allowedCommandIdentityV0 struct{}
type authority map[string]*allowedCommandIdentityV0
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 {
	clone := authority(r.commands)
	return clone[key]
}`,
		"iife": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 {
	return func(hidden map[string]*allowedCommandIdentityV0) *allowedCommandIdentityV0 {
		return hidden[key]
	}(r.commands)
}`,
		"campo_nombre_alterno": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { catalog map[string]*allowedCommandIdentityV0 }
func rogue(r registry, key string) *allowedCommandIdentityV0 { return r.catalog[key] }`,
		"inicializador_global": `package fixture
type allowedCommandIdentityV0 struct{}
var authority = map[string]*allowedCommandIdentityV0{}
var escaped = authority["runner"]`,
		"range_value": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { catalog map[string]*allowedCommandIdentityV0 }
func rogue(r registry) *allowedCommandIdentityV0 {
	for _, identity := range r.catalog { return identity }
	return nil
}`,
		"range_selector_assignment": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
type holder struct { id *allowedCommandIdentityV0 }
func rogue(r registry) *allowedCommandIdentityV0 {
	var h holder
	for _, h.id = range r.commands { return h.id }
	return nil
}`,
		"maps_all": `package fixture
import "maps"
type allowedCommandIdentityV0 struct{}
type registry struct { catalog map[string]*allowedCommandIdentityV0 }
func rogue(r registry) *allowedCommandIdentityV0 {
	for _, identity := range maps.All(r.catalog) { return identity }
	return nil
}`,
		"maps_values": `package fixture
import "maps"
type allowedCommandIdentityV0 struct{}
type registry struct { catalog map[string]*allowedCommandIdentityV0 }
func rogue(r registry) *allowedCommandIdentityV0 {
	for identity := range maps.Values(r.catalog) { return identity }
	return nil
}`,
	}
	for nombre, source := range casos {
		t.Run(nombre, func(t *testing.T) {
			sitios, err := analizarAutoridadAllowlistV0(map[string]string{"fixture.go": source})
			if err != nil {
				t.Fatal(err)
			}
			if len(sitios) == 0 {
				t.Fatalf("el guard no detecto la autoridad oculta por %s", nombre)
			}
		})
	}
}

func TestGuardAutoridadAllowlistV0RechazaReceiverSpoofV0(t *testing.T) {
	source := `package fixture
type allowedCommandIdentityV0 struct{}
type spoofallowedCommandIdentityRegistryV0 struct { commands map[string]*allowedCommandIdentityV0 }
func (r *spoofallowedCommandIdentityRegistryV0) resolveTokens(key string) *allowedCommandIdentityV0 {
	return r.commands[key]
}`
	sitios, err := analizarAutoridadAllowlistV0(map[string]string{"fixture.go": source})
	if err != nil {
		t.Fatal(err)
	}
	if len(sitios) != 1 {
		t.Fatalf("receiver spoof no detectado: %v", sitios)
	}
	if validarPropietariosAutoridadAllowlistV0(sitios) == nil {
		t.Fatal("receiver con sufijo autorizado por coincidencia parcial")
	}
}

func TestGuardAutoridadAllowlistV0RechazaQuintaReferencia(t *testing.T) {
	source := `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func rogue1(r registry, k string) *allowedCommandIdentityV0 { return r.commands[k] }
func rogue2(r registry, k string) *allowedCommandIdentityV0 { return r.commands[k] }
func rogue3(r registry, k string) *allowedCommandIdentityV0 { return r.commands[k] }
func rogue4(r registry, k string) *allowedCommandIdentityV0 { return r.commands[k] }
func rogue5(r registry, k string) *allowedCommandIdentityV0 { return r.commands[k] }`
	sitios, err := analizarAutoridadAllowlistV0(map[string]string{"fixture.go": source})
	if err != nil {
		t.Fatal(err)
	}
	if len(sitios) != 5 {
		t.Fatalf("se esperaban 5 referencias, detectadas %d: %v", len(sitios), sitios)
	}
	if validarPropietariosAutoridadAllowlistV0(sitios) == nil {
		t.Fatal("el guard acepto una quinta referencia no autorizada")
	}
}

func TestGuardAutoridadAllowlistV0IgnoraConfiguracionYChequeoDescartado(t *testing.T) {
	casos := map[string]string{
		"configuracion": `package fixture
type config struct { AllowedCommands map[string]string }
func validate(c config, key string) bool { _, ok := c.AllowedCommands[key]; return ok }`,
		"duplicado_descarta_identidad": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func duplicate(r registry, key string) bool { _, duplicate := r.commands[key]; return duplicate }`,
		"range_solo_claves": `package fixture
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func keys(r registry) string { for key := range r.commands { return key }; return "" }`,
		"maps_keys": `package fixture
import "maps"
type allowedCommandIdentityV0 struct{}
type registry struct { commands map[string]*allowedCommandIdentityV0 }
func keys(r registry) string { for key := range maps.Keys(r.commands) { return key }; return "" }`,
		"range_configuracion": `package fixture
type config struct { AllowedCommands map[string]string }
func values(c config) string { for _, value := range c.AllowedCommands { return value }; return "" }`,
	}
	for nombre, source := range casos {
		t.Run(nombre, func(t *testing.T) {
			sitios, err := analizarAutoridadAllowlistV0(map[string]string{"fixture.go": source})
			if err != nil {
				t.Fatal(err)
			}
			if len(sitios) != 0 {
				t.Fatalf("el guard confundio %s con autoridad ejecutable: %v", nombre, sitios)
			}
		})
	}
}

func analizarAutoridadAllowlistV0(sources map[string]string) ([]sitioAutoridadAllowlistV0, error) {
	fset := token.NewFileSet()
	nombres := make([]string, 0, len(sources))
	for nombre := range sources {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	files := make([]*ast.File, 0, len(nombres))
	fileNames := make(map[*ast.File]string, len(nombres))
	for _, nombre := range nombres {
		file, err := parser.ParseFile(fset, nombre, sources[nombre], parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parseando %s: %w", nombre, err)
		}
		files = append(files, file)
		fileNames[file] = nombre
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	config := types.Config{
		Importer:                 importer.Default(),
		DisableUnusedImportCheck: true,
		Error:                    func(error) {}, // imports locales pueden no estar instalados; los objetos del paquete siguen tipados
	}
	_, _ = config.Check("orquesta/allowlistguard", fset, files, info)
	analyzer := &analizadorAutoridadAllowlistV0{
		fset:       fset,
		files:      files,
		names:      fileNames,
		info:       info,
		tainted:    make(map[types.Object]bool),
		funcAlias:  make(map[types.Object]*types.Func),
		returnsMap: make(map[*types.Func]bool),
	}
	analyzer.seed()
	for i := 0; i < 64 && analyzer.propagate(); i++ {
	}
	return analyzer.sites(), nil
}

func (a *analizadorAutoridadAllowlistV0) seed() {
	for _, file := range a.files {
		ast.Inspect(file, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			obj := a.info.Defs[ident]
			if obj != nil && esMapaAllowlistV0(obj.Type()) {
				a.tainted[obj] = true
			}
			return true
		})
	}
}

func (a *analizadorAutoridadAllowlistV0) propagate() bool {
	changed := false
	for _, file := range a.files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.AssignStmt:
				for i := 0; i < len(n.Lhs) && i < len(n.Rhs); i++ {
					changed = a.propagateAssignment(n.Lhs[i], n.Rhs[i]) || changed
				}
			case *ast.ValueSpec:
				for i := 0; i < len(n.Names) && i < len(n.Values); i++ {
					changed = a.taintObjectFromExpr(a.info.Defs[n.Names[i]], n.Values[i]) || changed
				}
			case *ast.RangeStmt:
				if a.exprTainted(n.X) {
					changed = a.taintExprObject(n.Key) || changed
					changed = a.taintExprObject(n.Value) || changed
				}
			case *ast.CallExpr:
				sig := a.callSignature(n.Fun)
				if sig == nil {
					break
				}
				for i := 0; i < len(n.Args) && i < sig.Params().Len(); i++ {
					if a.exprTainted(n.Args[i]) && !a.tainted[sig.Params().At(i)] {
						a.tainted[sig.Params().At(i)] = true
						changed = true
					}
				}
			}
			return true
		})
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			obj, _ := a.info.Defs[fn.Name].(*types.Func)
			if obj == nil || a.returnsMap[obj] {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				ret, ok := node.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				for _, result := range ret.Results {
					if a.exprTainted(result) {
						a.returnsMap[obj] = true
						changed = true
					}
				}
				return true
			})
		}
	}
	return changed
}

func (a *analizadorAutoridadAllowlistV0) propagateAssignment(lhs, rhs ast.Expr) bool {
	changed := false
	if target := a.exprObject(lhs); target != nil {
		changed = a.taintObjectFromExpr(target, rhs) || changed
		if fn := a.callee(rhs); fn != nil {
			if old := a.funcAlias[target]; old != fn {
				a.funcAlias[target] = fn
				changed = true
			}
		}
	}
	if index, ok := unparenExprV0(lhs).(*ast.IndexExpr); ok && a.exprTainted(index.Index) && a.exprTainted(rhs) {
		changed = a.taintExprObject(index.X) || changed
	}
	return changed
}

func (a *analizadorAutoridadAllowlistV0) taintObjectFromExpr(obj types.Object, expr ast.Expr) bool {
	if obj == nil || a.tainted[obj] || !a.exprTainted(expr) {
		return false
	}
	a.tainted[obj] = true
	return true
}

func (a *analizadorAutoridadAllowlistV0) taintExprObject(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	obj := a.exprObject(expr)
	if obj == nil || a.tainted[obj] {
		return false
	}
	a.tainted[obj] = true
	return true
}

func (a *analizadorAutoridadAllowlistV0) exprTainted(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	expr = unparenExprV0(expr)
	if obj := a.exprObject(expr); obj != nil && a.tainted[obj] {
		return true
	}
	switch e := expr.(type) {
	case *ast.IndexExpr:
		return a.exprTainted(e.X)
	case *ast.CallExpr:
		if a.returnsMap[a.callee(e.Fun)] {
			return true
		}
		// Conservative authority propagation covers generic/external map
		// cloning, conversions, map-returning IIFEs and iterator signatures
		// such as maps.All/maps.Values that transport admitted identities.
		if transportaIdentidadAllowlistV0(a.info.TypeOf(e)) {
			for _, arg := range e.Args {
				if a.exprTainted(arg) {
					return true
				}
			}
		}
	}
	return false
}

func (a *analizadorAutoridadAllowlistV0) exprObject(expr ast.Expr) types.Object {
	switch e := unparenExprV0(expr).(type) {
	case *ast.Ident:
		if obj := a.info.Uses[e]; obj != nil {
			return obj
		}
		return a.info.Defs[e]
	case *ast.SelectorExpr:
		if selection := a.info.Selections[e]; selection != nil {
			return selection.Obj()
		}
		return a.info.Uses[e.Sel]
	}
	return nil
}

func (a *analizadorAutoridadAllowlistV0) callee(expr ast.Expr) *types.Func {
	if obj := a.exprObject(expr); obj != nil {
		if fn, ok := obj.(*types.Func); ok {
			return fn
		}
		return a.funcAlias[obj]
	}
	return nil
}

func (a *analizadorAutoridadAllowlistV0) callSignature(expr ast.Expr) *types.Signature {
	if fn := a.callee(expr); fn != nil {
		sig, _ := fn.Type().(*types.Signature)
		return sig
	}
	sig, _ := a.info.TypeOf(unparenExprV0(expr)).(*types.Signature)
	return sig
}

func (a *analizadorAutoridadAllowlistV0) sites() []sitioAutoridadAllowlistV0 {
	var result []sitioAutoridadAllowlistV0
	for _, file := range a.files {
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				if typed.Body == nil {
					continue
				}
				owner, identity := typed.Name.Name, "func."+typed.Name.Name
				if obj, ok := a.info.Defs[typed.Name].(*types.Func); ok {
					owner = obj.FullName()
					identity = identidadPropietarioAutoridadAllowlistV0(obj)
				}
				result = append(result, a.sitesInNode(file, typed.Body, owner, identity)...)
			case *ast.GenDecl:
				// Package-level initializers execute outside any FuncDecl and must
				// not become a blind spot for authority reads.
				result = append(result, a.sitesInNode(file, typed, "package initializer", "package.init")...)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Fichero != result[j].Fichero {
			return result[i].Fichero < result[j].Fichero
		}
		return result[i].Linea < result[j].Linea
	})
	return result
}

func (a *analizadorAutoridadAllowlistV0) sitesInNode(
	file *ast.File,
	node ast.Node,
	owner string,
	ownerIdentity string,
) []sitioAutoridadAllowlistV0 {
	writes := make(map[*ast.IndexExpr]bool)
	discardedIdentities := make(map[*ast.IndexExpr]bool)
	ast.Inspect(node, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			if index, ok := unparenExprV0(lhs).(*ast.IndexExpr); ok {
				writes[index] = true
			}
		}
		if len(assign.Lhs) == 2 && len(assign.Rhs) == 1 {
			blank, blankOK := unparenExprV0(assign.Lhs[0]).(*ast.Ident)
			index, indexOK := unparenExprV0(assign.Rhs[0]).(*ast.IndexExpr)
			if blankOK && blank.Name == "_" && indexOK {
				discardedIdentities[index] = true
			}
		}
		return true
	})
	var result []sitioAutoridadAllowlistV0
	ast.Inspect(node, func(node ast.Node) bool {
		if ranged, ok := node.(*ast.RangeStmt); ok && a.exprTainted(ranged.X) && rangeExtraeIdentidadAllowlistV0(a.info, ranged) {
			pos := a.fset.Position(ranged.For)
			result = append(result, sitioAutoridadAllowlistV0{a.names[file], owner, ownerIdentity, pos.Line})
			return true
		}
		index, ok := node.(*ast.IndexExpr)
		if !ok || writes[index] || discardedIdentities[index] || !esMapaAllowlistV0(a.info.TypeOf(index.X)) || !a.exprTainted(index.X) {
			return true
		}
		pos := a.fset.Position(index.Lbrack)
		result = append(result, sitioAutoridadAllowlistV0{a.names[file], owner, ownerIdentity, pos.Line})
		return true
	})
	return result
}

func identidadPropietarioAutoridadAllowlistV0(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	sig, _ := fn.Type().(*types.Signature)
	if sig == nil || sig.Recv() == nil {
		return "func." + fn.Name()
	}
	receiver := sig.Recv().Type()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	named, ok := receiver.(*types.Named)
	if !ok || named.Obj() == nil {
		return ""
	}
	return named.Obj().Name() + "." + fn.Name()
}

func validarPropietariosAutoridadAllowlistV0(sitios []sitioAutoridadAllowlistV0) error {
	if len(sitios) == 0 {
		return fmt.Errorf("no se encontro ninguna resolucion de la allowlist: ¿se ha desactivado el control?")
	}
	if len(sitios) > maxSitiosAutoridadAllowlistV0 {
		return fmt.Errorf(
			"la autoridad allowlist aparece en %d sitios (tope %d): %v",
			len(sitios), maxSitiosAutoridadAllowlistV0, sitios,
		)
	}
	count := 0
	for _, sitio := range sitios {
		if sitio.Propietario != propietarioAutoridadAllowlistV0 {
			return fmt.Errorf("autoridad allowlist no autorizada en %s:%d (%s)", sitio.Fichero, sitio.Linea, sitio.Funcion)
		}
		count++
		if count > 1 {
			return fmt.Errorf("%s resuelve la autoridad allowlist %d veces; el ratchet permite una", propietarioAutoridadAllowlistV0, count)
		}
	}
	return nil
}

func esMapaAllowlistV0(t types.Type) bool {
	if t == nil {
		return false
	}
	m, ok := t.Underlying().(*types.Map)
	if !ok {
		return false
	}
	ptr, ok := m.Elem().(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Name() == "allowedCommandIdentityV0"
}

func rangeExtraeIdentidadAllowlistV0(info *types.Info, ranged *ast.RangeStmt) bool {
	if ranged == nil {
		return false
	}
	for _, candidate := range []ast.Expr{ranged.Key, ranged.Value} {
		if candidate == nil {
			continue
		}
		if ident, ok := unparenExprV0(candidate).(*ast.Ident); ok && ident.Name == "_" {
			continue
		}
		if esIdentidadAllowlistV0(info.TypeOf(candidate)) {
			return true
		}
	}
	return false
}

func transportaIdentidadAllowlistV0(t types.Type) bool {
	return transportaIdentidadAllowlistVisitV0(t, make(map[types.Type]bool))
}

func transportaIdentidadAllowlistVisitV0(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	if esIdentidadAllowlistV0(t) {
		return true
	}
	switch typed := t.Underlying().(type) {
	case *types.Map:
		return transportaIdentidadAllowlistVisitV0(typed.Key(), seen) || transportaIdentidadAllowlistVisitV0(typed.Elem(), seen)
	case *types.Array:
		return transportaIdentidadAllowlistVisitV0(typed.Elem(), seen)
	case *types.Slice:
		return transportaIdentidadAllowlistVisitV0(typed.Elem(), seen)
	case *types.Chan:
		return transportaIdentidadAllowlistVisitV0(typed.Elem(), seen)
	case *types.Signature:
		return transportaIdentidadAllowlistTupleV0(typed.Params(), seen) || transportaIdentidadAllowlistTupleV0(typed.Results(), seen)
	case *types.Struct:
		for index := 0; index < typed.NumFields(); index++ {
			if transportaIdentidadAllowlistVisitV0(typed.Field(index).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func transportaIdentidadAllowlistTupleV0(tuple *types.Tuple, seen map[types.Type]bool) bool {
	if tuple == nil {
		return false
	}
	for index := 0; index < tuple.Len(); index++ {
		if transportaIdentidadAllowlistVisitV0(tuple.At(index).Type(), seen) {
			return true
		}
	}
	return false
}

func esIdentidadAllowlistV0(t types.Type) bool {
	pointer, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := pointer.Elem().(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Name() == "allowedCommandIdentityV0"
}

func unparenExprV0(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}
