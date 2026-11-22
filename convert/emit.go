package convert

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Options controls TypeSpec output formatting and modeling behavior.
type Options struct {
	RootName string
	Inline   bool
}

// ConvertJSONToTypeSpec parses JSON from r and returns the generated TypeSpec source.
func ConvertJSONToTypeSpec(r io.Reader, opts Options) (string, error) {
	val, err := parseOrderedJSON(r)
	if err != nil {
		return "", err
	}

	rootType := inferTypeNode(val)
	return renderTypeSpecDocument(rootType, opts), nil
}

type emittedModel struct {
	Name   string
	Fields []objectField
	Body   string
}

type typeSpecEmitter struct {
	inline     bool
	models     []emittedModel
	signatures map[string]string
}

func renderTypeSpecDocument(root typeNode, opts Options) string {
	rootName := normalizeModelName(cmp.Or(opts.RootName, "Root"))
	emitter := &typeSpecEmitter{
		inline:     opts.Inline,
		signatures: make(map[string]string),
	}

	var sections []string
	if root.Kind == typeObject {
		emitter.signatures[rootName] = computeObjectSignature(root.Fields)
		rootBody := emitter.formatModelDeclaration(rootName, root.Fields)
		sections = append(sections, rootBody)
	} else {
		expr := emitter.renderTypeExpr(root, rootName, rootName, 0)
		sections = append(sections, fmt.Sprintf("alias %s = %s;", rootName, expr))
	}

	for _, m := range emitter.models {
		sections = append(sections, m.Body)
	}

	return strings.Join(sections, "\n\n") + "\n"
}

func (e *typeSpecEmitter) formatModelDeclaration(name string, fields []objectField) string {
	if len(fields) == 0 {
		return fmt.Sprintf("model %s {}", name)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "model %s {\n", name)
	for _, field := range fields {
		propName := formatTypeSpecProperty(field.Name)
		optMark := ""
		if field.Optional {
			optMark = "?"
		}
		hint := normalizeModelName(field.Name)
		typeExpr := e.renderTypeExpr(field.Type, hint, name, 1)
		fmt.Fprintf(&b, "  %s%s: %s;\n", propName, optMark, typeExpr)
	}
	b.WriteByte('}')
	return b.String()
}

func (e *typeSpecEmitter) renderTypeExpr(node typeNode, nameHint, parentName string, indent int) string {
	switch node.Kind {
	case typeUnknown:
		return "unknown"
	case typeNull:
		return "null"
	case typeBool:
		return "boolean"
	case typeInt64:
		return "int64"
	case typeFloat64:
		return "float64"
	case typeString:
		return "string"
	case typePlainDate:
		return "plainDate"
	case typePlainTime:
		return "plainTime"
	case typeUTCDateTime:
		return "utcDateTime"
	case typeOffsetDateTime:
		return "offsetDateTime"
	case typeArray:
		elemHint := singularizeModelName(nameHint)
		elemExpr := e.renderTypeExpr(*node.Element, elemHint, parentName, indent)
		if node.Element.Kind == typeUnion {
			return "(" + elemExpr + ")[]"
		}
		return elemExpr + "[]"
	case typeUnion:
		parts := make([]string, 0, len(node.Members))
		for _, member := range node.Members {
			parts = append(parts, e.renderTypeExpr(member, nameHint, parentName, indent))
		}
		return strings.Join(parts, " | ")
	case typeObject:
		if e.inline {
			return e.renderInlineObject(node.Fields, nameHint, parentName, indent)
		}
		return e.registerExtractedModel(node.Fields, nameHint, parentName)
	default:
		return "unknown"
	}
}

func (e *typeSpecEmitter) renderInlineObject(fields []objectField, nameHint, parentName string, indent int) string {
	if len(fields) == 0 {
		return "{}"
	}

	var b strings.Builder
	b.WriteString("{\n")
	childIndent := strings.Repeat("  ", indent+1)
	closeIndent := strings.Repeat("  ", indent)
	for _, field := range fields {
		propName := formatTypeSpecProperty(field.Name)
		optMark := ""
		if field.Optional {
			optMark = "?"
		}
		fieldHint := normalizeModelName(field.Name)
		typeExpr := e.renderTypeExpr(field.Type, fieldHint, nameHint, indent+1)
		fmt.Fprintf(&b, "%s%s%s: %s;\n", childIndent, propName, optMark, typeExpr)
	}
	fmt.Fprintf(&b, "%s}", closeIndent)
	return b.String()
}

func (e *typeSpecEmitter) registerExtractedModel(fields []objectField, nameHint, parentName string) string {
	sig := computeObjectSignature(fields)
	candidate := normalizeModelName(nameHint)

	if existingSig, exists := e.signatures[candidate]; exists {
		if existingSig == sig {
			return candidate
		}
		prefixed := normalizeModelName(parentName) + candidate
		if prefixedSig, prefixedExists := e.signatures[prefixed]; prefixedExists {
			if prefixedSig == sig {
				return prefixed
			}
			candidate = e.allocateUniqueModelName(prefixed, sig)
			if e.signatures[candidate] == sig {
				return candidate
			}
		} else {
			candidate = prefixed
		}
	}

	e.signatures[candidate] = sig
	idx := len(e.models)
	e.models = append(e.models, emittedModel{Name: candidate, Fields: fields})
	e.models[idx].Body = e.formatModelDeclaration(candidate, fields)
	return candidate
}

func (e *typeSpecEmitter) allocateUniqueModelName(base, sig string) string {
	for suffix := 2; ; suffix++ {
		name := base + strconv.Itoa(suffix)
		existingSig, exists := e.signatures[name]
		if !exists || existingSig == sig {
			return name
		}
	}
}

func computeObjectSignature(fields []objectField) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, field := range fields {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(field.Name)
		if field.Optional {
			b.WriteByte('?')
		}
		b.WriteByte(':')
		writeTypeSignature(&b, field.Type)
	}
	b.WriteByte('}')
	return b.String()
}

func writeTypeSignature(b *strings.Builder, node typeNode) {
	switch node.Kind {
	case typeUnknown:
		b.WriteString("unknown")
	case typeNull:
		b.WriteString("null")
	case typeBool:
		b.WriteString("boolean")
	case typeInt64:
		b.WriteString("int64")
	case typeFloat64:
		b.WriteString("float64")
	case typeString:
		b.WriteString("string")
	case typePlainDate:
		b.WriteString("plainDate")
	case typePlainTime:
		b.WriteString("plainTime")
	case typeUTCDateTime:
		b.WriteString("utcDateTime")
	case typeOffsetDateTime:
		b.WriteString("offsetDateTime")
	case typeArray:
		b.WriteByte('[')
		writeTypeSignature(b, *node.Element)
		b.WriteByte(']')
	case typeUnion:
		b.WriteByte('(')
		for i, m := range node.Members {
			if i > 0 {
				b.WriteByte('|')
			}
			writeTypeSignature(b, m)
		}
		b.WriteByte(')')
	case typeObject:
		b.WriteString(computeObjectSignature(node.Fields))
	}
}

var typeSpecKeywords = []string{
	"alias", "const", "dec", "else", "enum", "extends", "extern",
	"false", "fn", "if", "import", "init", "interface", "internal",
	"is", "model", "namespace", "never", "null", "op", "projection",
	"return", "scalar", "true", "typeof", "union", "unknown", "using",
	"valueof", "void",
}

func formatTypeSpecProperty(name string) string {
	if isValidTypeSpecIdentifier(name) {
		return name
	}
	escaped := strings.ReplaceAll(name, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	return "`" + escaped + "`"
}

func isValidTypeSpecIdentifier(name string) bool {
	if name == "" || slices.Contains(typeSpecKeywords, name) {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isIdentifierStart(r) {
				return false
			}
			continue
		}
		if !isIdentifierPart(r) {
			return false
		}
	}
	return true
}

func isIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '$'
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || (r >= '0' && r <= '9')
}

func normalizeModelName(raw string) string {
	var b strings.Builder
	capitalizeNext := true
	for _, r := range raw {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			b.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			b.WriteRune(r)
		}
	}

	result := b.String()
	if result == "" {
		return "Model"
	}
	if unicode.IsDigit(rune(result[0])) {
		return "Model" + result
	}
	if slices.Contains(typeSpecKeywords, result) {
		return result + "Model"
	}
	return result
}

func singularizeModelName(name string) string {
	normalized := normalizeModelName(name)
	lower := strings.ToLower(normalized)

	switch {
	case len(normalized) > 3 && strings.HasSuffix(lower, "ies"):
		return normalized[:len(normalized)-3] + "y"
	case len(normalized) > 4 && (strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes")):
		return normalized[:len(normalized)-2]
	case len(normalized) > 3 && (strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") || strings.HasSuffix(lower, "zes")):
		return normalized[:len(normalized)-2]
	case len(normalized) > 1 && strings.HasSuffix(lower, "s") &&
		!strings.HasSuffix(lower, "ss") &&
		!strings.HasSuffix(lower, "us") &&
		!strings.HasSuffix(lower, "is"):
		return normalized[:len(normalized)-1]
	default:
		return normalized + "Item"
	}
}
