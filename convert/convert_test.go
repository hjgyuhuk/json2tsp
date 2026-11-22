package convert

import (
	"strings"
	"testing"
)

func TestConvertJSONToTypeSpec(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  Options
		want  string
	}{
		{
			name: "preserves key order and scalar types",
			input: `{
				"z_id": 42,
				"a_price": 19.99,
				"active": true,
				"note": null,
				"birth_date": "2026-11-22",
				"alarm_time": "18:32:22",
				"precise_time": "18:32:22.125",
				"created_at": "2026-11-22T18:32:22Z",
				"updated_at": "2026-11-22T18:32:22.123+08:00",
				"mixed_times": ["2026-11-22T10:00:00Z", "2026-11-22T18:00:00+08:00"],
				"title": "hello"
			}`,
			opts: Options{RootName: "Order"},
			want: strings.Join([]string{
				"model Order {",
				"  z_id: int64;",
				"  a_price: float64;",
				"  active: boolean;",
				"  note: null;",
				"  birth_date: plainDate;",
				"  alarm_time: plainTime;",
				"  precise_time: plainTime;",
				"  created_at: utcDateTime;",
				"  updated_at: offsetDateTime;",
				"  mixed_times: offsetDateTime[];",
				"  title: string;",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "extracts nested models by default",
			input: `{
				"id": 1,
				"user": {
					"name": "Alice",
					"address": {
						"city": "Tokyo"
					}
				}
			}`,
			opts: Options{RootName: "Response"},
			want: strings.Join([]string{
				"model Response {",
				"  id: int64;",
				"  user: User;",
				"}",
				"",
				"model User {",
				"  name: string;",
				"  address: Address;",
				"}",
				"",
				"model Address {",
				"  city: string;",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "inlines nested objects when inline enabled",
			input: `{
				"id": 1,
				"user": {
					"name": "Alice",
					"address": {
						"city": "Tokyo"
					}
				}
			}`,
			opts: Options{RootName: "Response", Inline: true},
			want: strings.Join([]string{
				"model Response {",
				"  id: int64;",
				"  user: {",
				"    name: string;",
				"    address: {",
				"      city: string;",
				"    };",
				"  };",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "merges object array fields and marks missing fields optional",
			input: `{
				"items": [
					{"id": 1, "price": 10, "tag": "2026-01-01"},
					{"id": 2, "price": 12.5, "tag": "promo", "discount": 0.1},
					{"id": 3, "price": 8, "note": null}
				]
			}`,
			opts: Options{},
			want: strings.Join([]string{
				"model Root {",
				"  items: Item[];",
				"}",
				"",
				"model Item {",
				"  id: int64;",
				"  price: float64;",
				"  tag?: string;",
				"  discount?: float64;",
				"  note?: null;",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "escapes invalid identifiers and keywords with backticks",
			input: `{
				"kebab-key": "v1",
				"123num": 1,
				"model": "reserved",
				"space key": true,
				"back` + "`" + `tick": "x"
			}`,
			opts: Options{RootName: "Special"},
			want: strings.Join([]string{
				"model Special {",
				"  `kebab-key`: string;",
				"  `123num`: int64;",
				"  `model`: string;",
				"  `space key`: boolean;",
				"  `back\\`tick`: string;",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "handles unions, nullable values, and empty arrays",
			input: `{
				"mixed": [1, "two", null],
				"empty": [],
				"records": [
					{"value": "abc"},
					{"value": null}
				]
			}`,
			opts: Options{RootName: "Payload"},
			want: strings.Join([]string{
				"model Payload {",
				"  mixed: (int64 | string | null)[];",
				"  empty: unknown[];",
				"  records: Record[];",
				"}",
				"",
				"model Record {",
				"  value: string | null;",
				"}",
				"",
			}, "\n"),
		},
		{
			name: "disambiguates colliding model names with different shapes and reuses identical shapes",
			input: `{
				"first": {
					"meta": {"a": 1}
				},
				"second": {
					"meta": {"b": "x"}
				},
				"third": {
					"meta": {"a": 2}
				}
			}`,
			opts: Options{RootName: "Doc"},
			want: strings.Join([]string{
				"model Doc {",
				"  first: First;",
				"  second: Second;",
				"  third: Third;",
				"}",
				"",
				"model First {",
				"  meta: Meta;",
				"}",
				"",
				"model Meta {",
				"  a: int64;",
				"}",
				"",
				"model Second {",
				"  meta: SecondMeta;",
				"}",
				"",
				"model SecondMeta {",
				"  b: string;",
				"}",
				"",
				"model Third {",
				"  meta: Meta;",
				"}",
				"",
			}, "\n"),
		},
		{
			name:  "top-level array of objects",
			input: `[{"id": 1}, {"id": 2, "name": "bob"}]`,
			opts:  Options{RootName: "Users"},
			want: strings.Join([]string{
				"alias Users = User[];",
				"",
				"model User {",
				"  id: int64;",
				"  name?: string;",
				"}",
				"",
			}, "\n"),
		},
		{
			name:  "top-level primitive",
			input: `"2026-11-22T18:32:22Z"`,
			opts:  Options{RootName: "Timestamp"},
			want:  "alias Timestamp = utcDateTime;\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertJSONToTypeSpec(strings.NewReader(tc.input), tc.opts)
			if err != nil {
				t.Fatalf("ConvertJSONToTypeSpec() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ConvertJSONToTypeSpec() mismatch:\n--- got ---\n%s--- want ---\n%s", got, tc.want)
			}
		})
	}
}

func TestConvertJSONToTypeSpecErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty input", input: ""},
		{name: "whitespace only", input: "   \n\t "},
		{name: "invalid json", input: `{"a": }`},
		{name: "trailing garbage", input: `{"a": 1} {"b": 2}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ConvertJSONToTypeSpec(strings.NewReader(tc.input), Options{})
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.input)
			}
		})
	}
}
