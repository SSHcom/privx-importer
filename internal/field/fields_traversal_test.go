package field

import (
	"testing"
)

func TestWalkFields(t *testing.T) {
	fields := Fields{
		"hostname": StringField{
			FieldName:  "hostname",
			FieldValue: "server1",
		},
		"services": CompositeField{
			path: "services",
			fields: Fields{
				"services.port": IntField{
					FieldName:  "services.port",
					FieldValue: 22,
				},
			},
		},
		"principals": ListCompositeField{
			path: "principals",
			items: []Fields{
				{
					"name": StringField{
						FieldName:  "principals.name",
						FieldValue: "root",
					},
				},
				{
					"name": StringField{
						FieldName:  "principals.name",
						FieldValue: "admin",
					},
				},
			},
		},
	}

	var visited []string

	err := fields.Walk(func(f Field) error {
		visited = append(visited, f.Name())
		return nil
	})
	if err != nil {
		t.Fatalf("WalkFields returned error: %v", err)
	}

	expectedCounts := map[string]int{
		"hostname":        1,
		"principals":      1,
		"principals.name": 2,
		"services":        1,
		"services.port":   1,
	}

	visitedCounts := make(map[string]int, len(expectedCounts))
	for _, name := range visited {
		visitedCounts[name]++
	}

	for name, want := range expectedCounts {
		got := visitedCounts[name]
		if got != want {
			t.Fatalf("visited count mismatch for %q: got %d want %d", name, got, want)
		}
	}
}

func TestTransformFields(t *testing.T) {
	fields := Fields{
		"hostname": StringField{
			FieldName:  "hostname",
			FieldValue: "server1",
		},
		"services": CompositeField{
			path: "services",
			fields: Fields{
				"services.port": IntField{
					FieldName:  "services.port",
					FieldValue: 22,
				},
			},
		},
	}

	transformed, err := fields.Transform(
		func(f Field) (Field, error) {
			if f.Name() == "hostname" {
				return f.Set("server2")
			}

			if f.Name() == "services.port" {
				return f.Set(2222)
			}

			return f, nil
		},
	)
	if err != nil {
		t.Fatalf("TransformFields returned error: %v", err)
	}

	hostname := transformed["hostname"].(StringField)
	if hostname.FieldValue != "server2" {
		t.Fatalf(
			"hostname mismatch: got %q want %q",
			hostname.FieldValue,
			"server2",
		)
	}

	services := transformed["services"].(CompositeField)

	port := services.fields["services.port"].(IntField)
	if port.FieldValue != 2222 {
		t.Fatalf(
			"port mismatch: got %d want %d",
			port.FieldValue,
			2222,
		)
	}
}

func TestTransformFieldsDoesNotModifyOriginal(t *testing.T) {
	fields := Fields{
		"hostname": StringField{
			FieldName:  "hostname",
			FieldValue: "server1",
		},
	}

	transformed, err := fields.Transform(
		func(f Field) (Field, error) {
			return f.Set("server2")
		},
	)
	if err != nil {
		t.Fatalf("TransformFields returned error: %v", err)
	}

	original := fields["hostname"].(StringField)
	if original.FieldValue != "server1" {
		t.Fatalf(
			"original modified: got %q want %q",
			original.FieldValue,
			"server1",
		)
	}

	updated := transformed["hostname"].(StringField)
	if updated.FieldValue != "server2" {
		t.Fatalf(
			"transformed mismatch: got %q want %q",
			updated.FieldValue,
			"server2",
		)
	}
}
