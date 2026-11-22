package convert

import (
	"slices"
	"strconv"
	"strings"
	"time"
)

type typeKind int

const (
	typeUnknown typeKind = iota
	typeNull
	typeBool
	typeInt64
	typeFloat64
	typeString
	typePlainDate
	typePlainTime
	typeUTCDateTime
	typeOffsetDateTime
	typeArray
	typeObject
	typeUnion
)

type objectField struct {
	Name     string
	Type     typeNode
	Optional bool
}

type typeNode struct {
	Kind    typeKind
	Element *typeNode
	Fields  []objectField
	Members []typeNode
}

func inferTypeNode(val jsonValue) typeNode {
	switch val.Kind {
	case jsonNull:
		return typeNode{Kind: typeNull}
	case jsonBool:
		return typeNode{Kind: typeBool}
	case jsonNumber:
		return typeNode{Kind: classifyNumberKind(val.Number)}
	case jsonString:
		return typeNode{Kind: classifyStringKind(val.String)}
	case jsonArray:
		return inferArrayType(val.Items)
	case jsonObject:
		return inferObjectType(val.Entries)
	default:
		return typeNode{Kind: typeUnknown}
	}
}

func classifyNumberKind(raw string) typeKind {
	if strings.ContainsAny(raw, ".eE") {
		return typeFloat64
	}
	if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return typeInt64
	}
	return typeFloat64
}

func classifyStringKind(raw string) typeKind {
	if len(raw) == len(time.DateOnly) {
		if _, err := time.Parse(time.DateOnly, raw); err == nil {
			return typePlainDate
		}
	}
	if len(raw) >= len(time.TimeOnly) && raw[2] == ':' && raw[5] == ':' {
		if _, err := time.Parse("15:04:05.999999999", raw); err == nil {
			return typePlainTime
		}
	}
	if len(raw) >= 20 && strings.Contains(raw, "T") {
		if _, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			if strings.HasSuffix(raw, "Z") {
				return typeUTCDateTime
			}
			return typeOffsetDateTime
		}
	}
	return typeString
}

func inferArrayType(items []jsonValue) typeNode {
	if len(items) == 0 {
		return typeNode{
			Kind:    typeArray,
			Element: new(typeNode{Kind: typeUnknown}),
		}
	}

	merged := inferTypeNode(items[0])
	for _, item := range items[1:] {
		merged = mergeTypeNodes(merged, inferTypeNode(item))
	}

	return typeNode{
		Kind:    typeArray,
		Element: new(merged),
	}
}

func inferObjectType(entries []jsonEntry) typeNode {
	fields := make([]objectField, 0, len(entries))
	for _, entry := range entries {
		fields = append(fields, objectField{
			Name: entry.Key,
			Type: inferTypeNode(entry.Value),
		})
	}
	return typeNode{
		Kind:   typeObject,
		Fields: fields,
	}
}

func mergeTypeNodes(left, right typeNode) typeNode {
	if left.Kind == typeUnknown {
		return right
	}
	if right.Kind == typeUnknown {
		return left
	}

	if merged, ok := tryMergeDirectly(left, right); ok {
		return merged
	}

	var members []typeNode
	members = appendMergedMember(members, left)
	members = appendMergedMember(members, right)
	members = moveNullMemberLast(members)

	if len(members) == 1 {
		return members[0]
	}
	return typeNode{Kind: typeUnion, Members: members}
}

func tryMergeDirectly(left, right typeNode) (typeNode, bool) {
	if left.Kind == right.Kind {
		switch left.Kind {
		case typeNull, typeBool, typeInt64, typeFloat64, typeString,
			typePlainDate, typePlainTime, typeUTCDateTime, typeOffsetDateTime:
			return left, true
		case typeArray:
			mergedElem := mergeTypeNodes(*left.Element, *right.Element)
			return typeNode{Kind: typeArray, Element: new(mergedElem)}, true
		case typeObject:
			return mergeObjectTypes(left, right), true
		}
	}

	if isNumericPair(left.Kind, right.Kind) {
		return typeNode{Kind: typeFloat64}, true
	}
	if isDateTimeWideningPair(left.Kind, right.Kind) {
		return typeNode{Kind: typeOffsetDateTime}, true
	}
	if isStringWideningPair(left.Kind, right.Kind) {
		return typeNode{Kind: typeString}, true
	}
	return typeNode{}, false
}

func isNumericPair(a, b typeKind) bool {
	return (a == typeInt64 && b == typeFloat64) || (a == typeFloat64 && b == typeInt64)
}

func isDateTimeWideningPair(a, b typeKind) bool {
	return (a == typeUTCDateTime && b == typeOffsetDateTime) || (a == typeOffsetDateTime && b == typeUTCDateTime)
}

func isStringWideningPair(a, b typeKind) bool {
	return (a == typeString && isTemporalKind(b)) || (b == typeString && isTemporalKind(a))
}

func isTemporalKind(k typeKind) bool {
	switch k {
	case typePlainDate, typePlainTime, typeUTCDateTime, typeOffsetDateTime:
		return true
	default:
		return false
	}
}

func mergeObjectTypes(left, right typeNode) typeNode {
	merged := make([]objectField, 0, len(left.Fields)+len(right.Fields))

	for _, lf := range left.Fields {
		rightIdx := slices.IndexFunc(right.Fields, func(rf objectField) bool {
			return rf.Name == lf.Name
		})
		if rightIdx >= 0 {
			rf := right.Fields[rightIdx]
			merged = append(merged, objectField{
				Name:     lf.Name,
				Type:     mergeTypeNodes(lf.Type, rf.Type),
				Optional: lf.Optional || rf.Optional,
			})
		} else {
			lf.Optional = true
			merged = append(merged, lf)
		}
	}

	for _, rf := range right.Fields {
		exists := slices.ContainsFunc(left.Fields, func(lf objectField) bool {
			return lf.Name == rf.Name
		})
		if !exists {
			rf.Optional = true
			merged = append(merged, rf)
		}
	}

	return typeNode{Kind: typeObject, Fields: merged}
}

func appendMergedMember(members []typeNode, incoming typeNode) []typeNode {
	if incoming.Kind == typeUnion {
		for _, sub := range incoming.Members {
			members = appendMergedMember(members, sub)
		}
		return members
	}

	for i := range len(members) {
		if combined, ok := tryMergeDirectly(members[i], incoming); ok {
			members = slices.Delete(members, i, i+1)
			return appendMergedMember(members, combined)
		}
	}

	return append(members, incoming)
}

func moveNullMemberLast(members []typeNode) []typeNode {
	nullIdx := slices.IndexFunc(members, func(m typeNode) bool {
		return m.Kind == typeNull
	})
	if nullIdx < 0 || nullIdx == len(members)-1 {
		return members
	}
	nullNode := members[nullIdx]
	members = slices.Delete(members, nullIdx, nullIdx+1)
	return append(members, nullNode)
}
