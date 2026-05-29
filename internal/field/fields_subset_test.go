package field

import "testing"

func TestFieldsEqualsSubset(t *testing.T) {
	fals := false
	tru := true
	tests := []struct {
		name  string
		left  Fields
		right Fields
		want  bool
	}{
		{
			name:  "empty other is subset",
			left:  Fields{},
			right: Fields{},
			want:  true,
		},
		{
			name: "empty other is subset of non-empty fields",
			left: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			right: Fields{},
			want:  true,
		},
		{
			name: "missing field is not subset",
			left: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"prod"}},
			},
			want: false,
		},
		{
			name: "equal string field is subset",
			left: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			right: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			want: true,
		},
		{
			name: "different string field is not subset",
			left: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			right: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "world"},
			},
			want: false,
		},
		{
			name: "equal bool field is subset",
			left: Fields{
				"disabled": BoolField{FieldName: "disabled", FieldValue: &tru},
			},
			right: Fields{
				"disabled": BoolField{FieldName: "disabled", FieldValue: &tru},
			},
			want: true,
		},
		{
			name: "different bool field is not subset",
			left: Fields{
				"disabled": BoolField{FieldName: "disabled", FieldValue: &fals},
			},
			right: Fields{
				"disabled": BoolField{FieldName: "disabled", FieldValue: &tru},
			},
			want: false,
		},
		{
			name: "same list is subset",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web"}},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web"}},
			},
			want: true,
		},
		{
			name: "shorter list is subset",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web", "ssh"}},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web"}},
			},
			want: true,
		},
		{
			name: "shorter list is subset with different order",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web", "ssh"}},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"ssh", "prod"}},
			},
			want: true,
		},
		{
			name: "list with missing item is not subset",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web"}},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "ssh"}},
			},
			want: false,
		},
		{
			name: "longer other list is not subset",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod"}},
			},
			right: Fields{
				"tags": ListField{path: "tags", list: []string{"prod", "web"}},
			},
			want: false,
		},
		{
			name: "multiple matching fields are subset",
			left: Fields{
				"comment":  StringField{FieldName: "comment", FieldValue: "hello"},
				"disabled": BoolField{FieldName: "disabled", FieldValue: &fals},
				"tags":     ListField{path: "tags", list: []string{"prod", "web", "ssh"}},
			},
			right: Fields{
				"comment":  StringField{FieldName: "comment", FieldValue: "hello"},
				"disabled": BoolField{FieldName: "disabled", FieldValue: &fals},
				"tags":     ListField{path: "tags", list: []string{"web"}},
			},
			want: true,
		},
		{
			name: "one non-matching field makes whole set not subset",
			left: Fields{
				"comment":  StringField{FieldName: "comment", FieldValue: "hello"},
				"disabled": BoolField{FieldName: "disabled", FieldValue: &fals},
				"tags":     ListField{path: "tags", list: []string{"prod", "web"}},
			},
			right: Fields{
				"comment":  StringField{FieldName: "comment", FieldValue: "hello"},
				"disabled": BoolField{FieldName: "disabled", FieldValue: &tru},
				"tags":     ListField{path: "tags", list: []string{"web"}},
			},
			want: false,
		},
		{
			name: "same map key with different field kind is not subset",
			left: Fields{
				"tags": ListField{path: "tags", list: []string{"prod"}},
			},
			right: Fields{
				"tags": StringField{FieldName: "tags", FieldValue: "prod"},
			},
			want: false,
		},
		{
			name:  "nil receiver with empty other is subset",
			left:  nil,
			right: Fields{},
			want:  true,
		},
		{
			name: "nil receiver with non-empty other is not subset",
			left: nil,
			right: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			want: false,
		},
		{
			name: "nil other is subset",
			left: Fields{
				"comment": StringField{FieldName: "comment", FieldValue: "hello"},
			},
			right: nil,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.left.EqualsSubset(tt.right)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
