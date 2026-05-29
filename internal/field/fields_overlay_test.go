package field

import (
	"slices"
	"testing"
)

func TestFieldsOverlayIdenticalCompositeAndListComposite(t *testing.T) {
	current := testOverlayFields("server1", 22, []string{"root"})
	spec := testOverlayFields("server1", 22, []string{"root"})

	next, changed, _ := current.Overlay(spec, func(f Field) bool { return true })

	assertChanged(t, changed, nil)
	assertFieldEquals(t, next["hostname"], spec["hostname"])
	assertFieldEquals(t, next["services"], spec["services"])
	assertFieldEquals(t, next["principals"], spec["principals"])
}

func TestFieldsOverlayChangedCompositeField(t *testing.T) {
	current := testOverlayFields("server1", 22, []string{"root"})
	spec := testOverlayFields("server1", 2222, []string{"root"})

	next, changed, _ := current.Overlay(spec, func(f Field) bool { return true })

	assertChanged(t, changed, []string{"services.port"})
	assertFieldEquals(t, next["services"], spec["services"])
}

func TestFieldsOverlayChangedListCompositeField(t *testing.T) {
	current := testOverlayFields("server1", 22, []string{"root"})
	spec := testOverlayFields("server1", 22, []string{"admin"})

	next, changed, _ := current.Overlay(spec, func(f Field) bool { return true })

	assertChanged(t, changed, []string{"principals.principal"})
	assertFieldEquals(t, next["principals"], spec["principals"])
}

func TestFieldsOverlayListCompositeWithTwoValues(t *testing.T) {
	current := testOverlayFields("server1", 22, []string{"root", "deploy"})
	spec := testOverlayFields("server1", 22, []string{"root", "deploy"})

	next, changed, _ := current.Overlay(spec, func(f Field) bool { return true })

	assertChanged(t, changed, nil)
	assertFieldEquals(t, next["principals"], spec["principals"])
}

func testOverlayFields(hostname string, port int, principals []string) Fields {
	return Fields{
		"hostname": StringField{
			FieldName:  "hostname",
			FieldValue: hostname,
		},
		"services": CompositeField{
			path: "services",
			fields: Fields{
				"services.service": StringField{
					FieldName:  "services.service",
					FieldValue: "ssh",
				},
				"services.port": IntField{
					FieldName:  "services.port",
					FieldValue: port,
				},
			},
		},
		"principals": testOverlayPrincipals(principals),
	}
}

func testOverlayPrincipals(names []string) ListCompositeField {
	items := make([]Fields, 0, len(names))

	for _, name := range names {
		items = append(items, Fields{
			"principals.principal": StringField{
				FieldName:  "principals.principal",
				FieldValue: name,
			},
			"principals.roles": ListField{
				path: "principals.roles",
				list: []string{"admin"},
			},
		})
	}

	return ListCompositeField{
		path:  "principals",
		items: items,
	}
}

func assertChanged(t *testing.T, got []string, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("changed mismatch: got %v, want %v", got, want)
	}
}

func assertFieldEquals(t *testing.T, got Field, want Field) {
	t.Helper()

	if got == nil {
		t.Fatalf("field mismatch: got nil, want %v", want)
	}

	if !got.Equals(want) {
		t.Fatalf("field mismatch: got %v, want %v", got, want)
	}
}

func TestFieldsOverlayCompositeContainingListCompositeWithTwoValues(t *testing.T) {
	current := testOverlayFieldsWithApplications("server1", []string{"shell", "sftp"})
	spec := testOverlayFieldsWithApplications("server1", []string{"shell", "sftp"})

	next, changed, _ := current.Overlay(spec, func(f Field) bool { return true })

	assertChanged(t, changed, nil)
	assertFieldEquals(t, next["principals"], spec["principals"])
}

func testOverlayFieldsWithApplications(hostname string, applications []string) Fields {
	return Fields{
		"hostname": StringField{
			FieldName:  "hostname",
			FieldValue: hostname,
		},
		"principals": CompositeField{
			path: "principals",
			fields: Fields{
				"principals.principal": StringField{
					FieldName:  "principals.principal",
					FieldValue: "root",
				},
				"principals.applications": testOverlayApplications(applications),
			},
		},
	}
}

func testOverlayApplications(names []string) ListCompositeField {
	items := make([]Fields, 0, len(names))

	for _, name := range names {
		items = append(items, Fields{
			"principals.applications.name": StringField{
				FieldName:  "principals.applications.name",
				FieldValue: name,
			},
		})
	}

	return ListCompositeField{
		path:  "principals.applications",
		items: items,
	}
}
