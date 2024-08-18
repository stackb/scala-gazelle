package semanticdb

import (
	"log"
	"sort"
	"strings"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

func NewTextDocumentVisitor() *TextDocumentVisitor {
	return &TextDocumentVisitor{
		symbols: make(map[string]*spb.SymbolInformation),
		types:   make(map[string]*spb.Type),
	}
}

type TextDocumentVisitor struct {
	types   map[string]*spb.Type
	symbols map[string]*spb.SymbolInformation
	debug   bool
}

func toImport(symbol string) string {
	if strings.HasPrefix(symbol, "local") {
		return ""
	}
	if idx := strings.Index(symbol, "#"); idx != -1 {
		symbol = symbol[:idx]
	}
	if idx := strings.Index(symbol, "."); idx != -1 {
		symbol = symbol[:idx]
	}
	symbol = strings.ReplaceAll(symbol, "/", ".")
	return symbol
}

func (v *TextDocumentVisitor) File(filename string) *sppb.File {
	file := new(sppb.File)
	imports := make([]string, 0, len(v.types))
	seen := make(map[string]bool)
	for name := range v.types {
		imp := toImport(name)
		if imp == "" {
			continue
		}
		if _, exists := seen[imp]; exists {
			continue
		}
		seen[imp] = true
		imports = append(imports, imp)
		if v.debug {
			log.Println(filename, "import:", imp)
		}
	}
	sort.Strings(imports)
	file.Imports = imports
	return file
}

func (v *TextDocumentVisitor) addType(symbol string, node *spb.Type) {
	if _, ok := v.types[symbol]; ok {
		if v.debug {
			log.Printf("duplicate type registration: %s", symbol)
		}
		return
	}
	v.types[symbol] = node
}

func (v *TextDocumentVisitor) VisitTextDocument(node *spb.TextDocument) {
	for _, child := range node.Symbols {
		v.VisitSymbolInformation(child)
	}
	// TODO: occurrences? diagnostics? synthetics?
}

func (v *TextDocumentVisitor) VisitSymbolInformation(node *spb.SymbolInformation) {
	if _, ok := v.symbols[node.Symbol]; ok {
		return // already processed
	}
	v.symbols[node.Symbol] = node

	for _, child := range node.Annotations {
		v.VisitAnnotation(child)
	}

	v.VisitSignature(node.Signature)

	// TODO: what are overridden symbols
	//
	// for _, child := range node.OverriddenSymbols {
	// 	v.VisitOverridden(child)
	// }
}

func (v *TextDocumentVisitor) VisitScope(node *spb.Scope) {
	if node == nil {
		return
	}
	for _, child := range node.Hardlinks {
		v.VisitSymbolInformation(child)
	}
	for _, child := range node.Symlinks {
		v.VisitSymlink(child)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitAnnotation(node *spb.Annotation) {
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitSymlink(name string) {
	node, ok := v.symbols[name]
	if !ok {
		if v.debug {
			log.Println("unknown symlink:", name)
		}
		v.addType(name, nil)
		return
	}
	v.VisitSymbolInformation(node)
	// DONE
}

func (v *TextDocumentVisitor) VisitSignature(node *spb.Signature) {
	switch t := node.SealedValue.(type) {
	case *spb.Signature_ClassSignature:
		v.VisitClassSignature(t.ClassSignature)
	case *spb.Signature_MethodSignature:
		v.VisitMethodSignature(t.MethodSignature)
	case *spb.Signature_TypeSignature:
		v.VisitTypeSignature(t.TypeSignature)
	case *spb.Signature_ValueSignature:
		v.VisitValueSignature(t.ValueSignature)
	default:
		log.Panicf("unexpected signature type: %T", t)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitClassSignature(node *spb.ClassSignature) {
	v.VisitScope(node.TypeParameters)
	for _, child := range node.Parents {
		v.VisitType(child)
	}
	v.VisitType(node.Self)
	v.VisitScope(node.Declarations)
	// DONE
}

func (v *TextDocumentVisitor) VisitMethodSignature(node *spb.MethodSignature) {
	v.VisitScope(node.TypeParameters)
	for _, child := range node.ParameterLists {
		v.VisitScope(child)
	}
	v.VisitType(node.ReturnType)
	// DONE
}

func (v *TextDocumentVisitor) VisitTypeSignature(node *spb.TypeSignature) {
	v.VisitScope(node.TypeParameters)
	v.VisitType(node.LowerBound)
	v.VisitType(node.UpperBound)
	// DONE
}

func (v *TextDocumentVisitor) VisitValueSignature(node *spb.ValueSignature) {
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitType(node *spb.Type) {
	if node == nil {
		return
	}

	switch t := node.SealedValue.(type) {
	case *spb.Type_TypeRef:
		v.VisitTypeRef(t.TypeRef)
	case *spb.Type_SingleType:
		v.VisitSingleType(node, t.SingleType)
	case *spb.Type_ThisType:
		v.VisitThisType(node, t.ThisType)
	case *spb.Type_SuperType:
		v.VisitSuperType(node, t.SuperType)
	case *spb.Type_ConstantType:
		v.VisitConstantType(t.ConstantType)
	case *spb.Type_IntersectionType:
		v.VisitIntersectionType(t.IntersectionType)
	case *spb.Type_UnionType:
		v.VisitUnionType(t.UnionType)
	case *spb.Type_WithType:
		v.VisitWithType(t.WithType)
	case *spb.Type_StructuralType:
		v.VisitStructuralType(t.StructuralType)
	case *spb.Type_AnnotatedType:
		v.VisitAnnotatedType(t.AnnotatedType)
	case *spb.Type_ExistentialType:
		v.VisitExistentialType(t.ExistentialType)
	case *spb.Type_UniversalType:
		v.VisitUniversalType(t.UniversalType)
	case *spb.Type_ByNameType:
		v.VisitByNameType(t.ByNameType)
	case *spb.Type_RepeatedType:
		v.VisitRepeatedType(t.RepeatedType)
	case *spb.Type_MatchType:
		v.VisitMatchType(t.MatchType)
	case *spb.Type_LambdaType:
		v.VisitLambdaType(t.LambdaType)
	default:
		log.Panicf("unexpected type: %T", t)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitTypeRef(node *spb.TypeRef) {
	if node == nil {
		return
	}

	v.VisitType(node.Prefix)
	for _, child := range node.TypeArguments {
		v.VisitType(child)
	}
	if _, ok := v.types[node.Symbol]; !ok {
		if v.debug {
			log.Printf("unknown typeref target: %s", node.Symbol)
		}
		v.addType(node.Symbol, nil)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitSingleType(parent *spb.Type, node *spb.SingleType) {
	v.addType(node.Symbol, parent)
	v.VisitType(node.Prefix)
	// DONE
}

func (v *TextDocumentVisitor) VisitThisType(parent *spb.Type, node *spb.ThisType) {
	v.addType(node.Symbol, parent)
	// DONE
}

func (v *TextDocumentVisitor) VisitSuperType(parent *spb.Type, node *spb.SuperType) {
	v.addType(node.Symbol, parent)
	v.VisitType(node.Prefix)
	// DONE
}

func (v *TextDocumentVisitor) VisitConstantType(node *spb.ConstantType) {
	v.VisitConstant(node.Constant)
	// DONE
}

func (v *TextDocumentVisitor) VisitIntersectionType(node *spb.IntersectionType) {
	for _, child := range node.Types {
		v.VisitType(child)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitUnionType(node *spb.UnionType) {
	for _, child := range node.Types {
		v.VisitType(child)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitWithType(node *spb.WithType) {
	for _, child := range node.Types {
		v.VisitType(child)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitStructuralType(node *spb.StructuralType) {
	v.VisitType(node.Tpe)
	v.VisitScope(node.Declarations)
	// DONE
}

func (v *TextDocumentVisitor) VisitAnnotatedType(node *spb.AnnotatedType) {
	for _, child := range node.Annotations {
		v.VisitAnnotation(child)
	}
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitExistentialType(node *spb.ExistentialType) {
	v.VisitType(node.Tpe)
	v.VisitScope(node.Declarations)
	// DONE
}

func (v *TextDocumentVisitor) VisitUniversalType(node *spb.UniversalType) {
	v.VisitScope(node.TypeParameters)
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitByNameType(node *spb.ByNameType) {
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitRepeatedType(node *spb.RepeatedType) {
	v.VisitType(node.Tpe)
	// DONE
}

func (v *TextDocumentVisitor) VisitMatchType(node *spb.MatchType) {
	v.VisitType(node.Scrutinee)
	for _, child := range node.Cases {
		v.VisitMatchCase(child)
	}
	// DONE
}

func (v *TextDocumentVisitor) VisitLambdaType(node *spb.LambdaType) {
	v.VisitScope(node.Parameters)
	v.VisitType(node.ReturnType)
	// DONE
}

func (v *TextDocumentVisitor) VisitMatchCase(node *spb.MatchType_CaseType) {
	v.VisitType(node.Key)
	v.VisitType(node.Body)
	// DONE
}

func (v *TextDocumentVisitor) VisitConstant(node *spb.Constant) {
	// DONE - no need to visit constants, right?
}
