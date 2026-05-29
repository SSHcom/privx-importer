package util

// Enum defines allowed string values for a typed configuration option.
type Enum[T ~string] map[string]T

// Values returns allowed enum values in stable order.
func (e Enum[T]) Values() []string {
	return Keys(e)
}

// Parse parses a string into a typed enum value.
func (e Enum[T]) Parse(value string) (T, error) {
	normalized, err := NormalizeToList(value, e.Values())
	if err != nil {
		var zero T
		return zero, err
	}

	return e[normalized], nil
}
