package field

import "testing"

func TestListCompositeFieldEquals(t *testing.T) {
	tests := []struct {
		name  string
		left  ListCompositeField
		right Field
		want  bool
	}{
		{
			name: "equal single item",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			want: true,
		},
		{
			name: "equal items in different order",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
				compositeItem(
					stringField("services.service", "rdp"),
					intField("services.port", 3389),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "rdp"),
					intField("services.port", 3389),
				),
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			want: false,
		},
		{
			name: "different item value",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 2222),
				),
			),
			want: false,
		},
		{
			name: "different item count",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
				compositeItem(
					stringField("services.service", "rdp"),
					intField("services.port", 3389),
				),
			),
			want: false,
		},
		{
			name: "duplicate item count matters",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			want: false,
		},
		{
			name: "different field type",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
				),
			),
			right: stringField("services", "ssh"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.left.Equals(tt.right)
			if got != tt.want {
				t.Fatalf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListCompositeFieldEqualsSubset(t *testing.T) {
	tests := []struct {
		name  string
		left  ListCompositeField
		right Field
		want  bool
	}{
		{
			name: "right side item subset matches",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
				),
			),
			want: true,
		},
		{
			name: "right side list subset matches",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
				compositeItem(
					stringField("services.service", "rdp"),
					intField("services.port", 3389),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "rdp"),
				),
			),
			want: true,
		},
		{
			name: "right side has missing item",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
					intField("services.port", 22),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "rdp"),
				),
			),
			want: false,
		},
		{
			name: "right side has more matching duplicates than left",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
				),
			),
			right: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
				),
				compositeItem(
					stringField("services.service", "ssh"),
				),
			),
			want: false,
		},
		{
			name: "different field type",
			left: listComposite("services",
				compositeItem(
					stringField("services.service", "ssh"),
				),
			),
			right: stringField("services", "ssh"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.left.EqualsSubset(tt.right)
			if got != tt.want {
				t.Fatalf("EqualsSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func listComposite(path string, items ...Fields) ListCompositeField {
	return ListCompositeField{
		path:  path,
		items: items,
	}
}

func compositeItem(fields ...Field) Fields {
	item := Fields{}

	for _, f := range fields {
		item[f.Name()] = f
	}

	return item
}

func stringField(name string, value string) StringField {
	return StringField{
		FieldName:  name,
		FieldValue: value,
	}
}

func intField(name string, value int) IntField {
	return IntField{
		FieldName:  name,
		FieldValue: value,
	}
}
