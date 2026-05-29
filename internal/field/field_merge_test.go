package field

import (
	"errors"
	"testing"
)

func TestStringFieldMerge(t *testing.T) {
	tests := []struct {
		name    string
		field   StringField
		value   any
		want    string
		wantErr error
	}{
		{
			name:  "empty field takes value",
			field: StringField{FieldName: "comment"},
			value: "new value",
			want:  "new value",
		},
		{
			name:  "empty value keeps existing",
			field: StringField{FieldName: "comment", FieldValue: "existing"},
			value: "",
			want:  "existing",
		},
		{
			name:  "both empty stays empty",
			field: StringField{FieldName: "comment"},
			value: "",
			want:  "",
		},
		{
			name:  "same value succeeds",
			field: StringField{FieldName: "comment", FieldValue: "same"},
			value: "same",
			want:  "same",
		},
		{
			name:    "different non-empty value conflicts",
			field:   StringField{FieldName: "comment", FieldValue: "old"},
			value:   "new",
			wantErr: ErrScalarMergeConflict,
		},
		{
			name:    "wrong value type fails",
			field:   StringField{FieldName: "comment", FieldValue: "old"},
			value:   123,
			wantErr: ErrFieldValueTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.Merge(tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sf, ok := got.(StringField)
			if !ok {
				t.Fatalf("expected StringField, got %T", got)
			}

			if sf.FieldValue != tt.want {
				t.Fatalf("expected value %q, got %q", tt.want, sf.FieldValue)
			}
		})
	}
}

func TestBoolFieldMerge(t *testing.T) {
	fals := false
	tru := true
	tests := []struct {
		name    string
		field   BoolField
		value   any
		want    bool
		wantErr error
	}{

		{
			name:  "same false succeeds",
			field: BoolField{FieldName: "disabled", FieldValue: &fals},
			value: false,
			want:  false,
		},
		{
			name:  "same true succeeds",
			field: BoolField{FieldName: "disabled", FieldValue: &tru},
			value: true,
			want:  true,
		},
		{
			name:    "different bool conflicts",
			field:   BoolField{FieldName: "disabled", FieldValue: &fals},
			value:   true,
			wantErr: ErrScalarMergeConflict,
		},
		{
			name:    "wrong value type fails",
			field:   BoolField{FieldName: "disabled", FieldValue: &fals},
			value:   "false",
			wantErr: ErrFieldValueTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.Merge(tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			bf, ok := got.(BoolField)
			if !ok {
				t.Fatalf("expected BoolField, got %T", got)
			}

			if *bf.FieldValue != tt.want {
				t.Fatalf("expected value %v, got %v", tt.want, bf.FieldValue)
			}
		})
	}
}

func TestListFieldMerge(t *testing.T) {
	tests := []struct {
		name    string
		field   ListField
		value   any
		want    []string
		wantErr error
	}{
		{
			name:  "empty field takes list",
			field: ListField{path: "tags"},
			value: []string{"prod", "web"},
			want:  []string{"prod", "web"},
		},
		{
			name:  "empty value keeps existing list",
			field: ListField{path: "tags", list: []string{"prod", "web"}},
			value: []string{},
			want:  []string{"prod", "web"},
		},
		{
			name:  "both empty stays empty",
			field: ListField{path: "tags"},
			value: []string{},
			want:  []string{},
		},
		{
			name:  "same list succeeds",
			field: ListField{path: "tags", list: []string{"prod", "web"}},
			value: []string{"prod", "web"},
			want:  []string{"prod", "web"},
		},
		{
			name:  "same list different order succeeds",
			field: ListField{path: "tags", list: []string{"prod", "web"}},
			value: []string{"web", "prod"},
			want:  []string{"prod", "web"},
		},
		{
			name:  "incoming list has extra item",
			field: ListField{path: "tags", list: []string{"prod", "web"}},
			value: []string{"prod", "web", "ssh"},
			want:  []string{"prod", "web", "ssh"},
		},
		{
			name:  "existing list has extra item",
			field: ListField{path: "tags", list: []string{"prod", "web", "ssh"}},
			value: []string{"prod", "web"},
			want:  []string{"prod", "web", "ssh"},
		},
		{
			name:  "both lists have different extra items",
			field: ListField{path: "tags", list: []string{"prod", "web"}},
			value: []string{"prod", "ssh"},
			want:  []string{"prod", "web", "ssh"},
		},
		{
			name:  "duplicate incoming values are deduplicated",
			field: ListField{path: "tags", list: []string{"prod"}},
			value: []string{"prod", "prod", "web"},
			want:  []string{"prod", "web"},
		},
		{
			name:  "string input becomes single element list",
			field: ListField{path: "tags"},
			value: "prod",
			want:  []string{"prod"},
		},
		{
			name:  "string merges into existing list",
			field: ListField{path: "tags", list: []string{"web"}},
			value: "prod",
			want:  []string{"web", "prod"},
		},
		{
			name:  "string duplicate is ignored",
			field: ListField{path: "tags", list: []string{"prod"}},
			value: "prod",
			want:  []string{"prod"},
		},
		{
			name:  "empty string is treated as empty list (no change)",
			field: ListField{path: "tags", list: []string{"prod"}},
			value: "",
			want:  []string{"prod"},
		},
		{
			name:  "empty field and empty string stays empty",
			field: ListField{path: "tags"},
			value: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.field.Merge(tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			lf, ok := got.(ListField)
			if !ok {
				t.Fatalf("expected ListField, got %T", got)
			}

			if !sameStringSet(lf.list, tt.want) {
				t.Fatalf("expected value %v, got %v", tt.want, lf.list)
			}
		})
	}
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[string]int, len(a))
	for _, v := range a {
		counts[v]++
	}

	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}

	return true
}
