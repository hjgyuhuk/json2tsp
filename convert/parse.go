package convert

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

type jsonKind int

const (
	jsonNull jsonKind = iota
	jsonBool
	jsonNumber
	jsonString
	jsonArray
	jsonObject
)

type jsonEntry struct {
	Key   string
	Value jsonValue
}

type jsonValue struct {
	Kind    jsonKind
	Bool    bool
	Number  string
	String  string
	Items   []jsonValue
	Entries []jsonEntry
}

func parseOrderedJSON(r io.Reader) (jsonValue, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return jsonValue{}, fmt.Errorf("empty json input: %w", err)
		}
		return jsonValue{}, fmt.Errorf("decode json root: %w", err)
	}

	val, err := decodeTokenValue(dec, tok)
	if err != nil {
		return jsonValue{}, err
	}

	if extra, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return jsonValue{}, fmt.Errorf("invalid trailing json content: %w", err)
		}
		return jsonValue{}, fmt.Errorf("unexpected trailing json token %v", extra)
	}

	return val, nil
}

func decodeTokenValue(dec *json.Decoder, tok json.Token) (jsonValue, error) {
	switch v := tok.(type) {
	case nil:
		return jsonValue{Kind: jsonNull}, nil
	case bool:
		return jsonValue{Kind: jsonBool, Bool: v}, nil
	case json.Number:
		return jsonValue{Kind: jsonNumber, Number: string(v)}, nil
	case string:
		return jsonValue{Kind: jsonString, String: v}, nil
	case json.Delim:
		switch v {
		case '{':
			return decodeOrderedObject(dec)
		case '[':
			return decodeOrderedArray(dec)
		default:
			return jsonValue{}, fmt.Errorf("unexpected json delimiter %q", v)
		}
	default:
		return jsonValue{}, fmt.Errorf("unsupported json token %T", tok)
	}
}

func decodeOrderedObject(dec *json.Decoder) (jsonValue, error) {
	var entries []jsonEntry

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return jsonValue{}, fmt.Errorf("decode json object key: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return jsonValue{}, fmt.Errorf("expected string key, got %T", keyTok)
		}

		valTok, err := dec.Token()
		if err != nil {
			return jsonValue{}, fmt.Errorf("decode json value for key %q: %w", key, err)
		}
		val, err := decodeTokenValue(dec, valTok)
		if err != nil {
			return jsonValue{}, err
		}

		existingIdx := slices.IndexFunc(entries, func(e jsonEntry) bool {
			return e.Key == key
		})
		if existingIdx >= 0 {
			entries[existingIdx].Value = val
		} else {
			entries = append(entries, jsonEntry{Key: key, Value: val})
		}
	}

	closeTok, err := dec.Token()
	if err != nil {
		return jsonValue{}, fmt.Errorf("decode json object end: %w", err)
	}
	if delim, ok := closeTok.(json.Delim); !ok || delim != '}' {
		return jsonValue{}, fmt.Errorf("expected closing '}', got %v", closeTok)
	}

	return jsonValue{Kind: jsonObject, Entries: entries}, nil
}

func decodeOrderedArray(dec *json.Decoder) (jsonValue, error) {
	var items []jsonValue

	for dec.More() {
		itemTok, err := dec.Token()
		if err != nil {
			return jsonValue{}, fmt.Errorf("decode json array item: %w", err)
		}
		item, err := decodeTokenValue(dec, itemTok)
		if err != nil {
			return jsonValue{}, err
		}
		items = append(items, item)
	}

	closeTok, err := dec.Token()
	if err != nil {
		return jsonValue{}, fmt.Errorf("decode json array end: %w", err)
	}
	if delim, ok := closeTok.(json.Delim); !ok || delim != ']' {
		return jsonValue{}, fmt.Errorf("expected closing ']', got %v", closeTok)
	}

	return jsonValue{Kind: jsonArray, Items: items}, nil
}
