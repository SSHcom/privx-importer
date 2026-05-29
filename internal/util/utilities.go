// Package util contains self-sufficient helper functions that do not import anything from other packages.
//
// These include
// string manipulation,
// array searches,
// map index helpers
package util

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var (
	// ErrValueNotFound indicates that value was not found from allowed values.
	ErrValueNotFound = errors.New("value not found")
)

// SortedKeys returns map keys in lexical order.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// FilterKeys returns a new map containing only allowed keys.
func FilterKeys[V any](input map[string]V, allowed []string) map[string]V {
	allowedSet := make(map[string]struct{}, len(allowed))

	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}

	result := make(map[string]V)

	for key, value := range input {
		if _, ok := allowedSet[key]; ok {
			result[key] = value
		}
	}

	return result
}

// Keys returns map keys as a sorted slice.
func Keys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	slices.Sort(keys)
	return keys
}

// NormalizeToList returns the canonical value from allowed values using
// case-insensitive matching.
func NormalizeToList(value string, allowed []string) (string, error) {
	for _, item := range allowed {
		if strings.EqualFold(value, item) {
			return item, nil
		}
	}

	return "", fmt.Errorf(
		"%w: %q is not one of allowed values: %s",
		ErrValueNotFound,
		value,
		strings.Join(allowed, ", "),
	)
}

// SplitCommaSeparated splits a comma separated list and returns an array of the trimmed elements.
func SplitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")

	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		out = append(out, part)
	}

	return out
}

// MaskString will mask partially the contents of a string.
func MaskString(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 15 {
		return "*****"
	}

	return value[:5] + "..." + value[len(value)-5:]
}

// StripIndexes removes list indexes like [0] from a path.
func StripIndexes(path string) string {
	re := regexp.MustCompile(`\[\d+\]`)
	return re.ReplaceAllString(path, "")
}

// Leaf returns the last segment of a canonical path "trunk.branch.branch.leaf".
func Leaf(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return path
	}

	return path[idx+1:]
}

// Trunk removes the segment of a canonical path "trunk.branch.branch.leaf".
func Trunk(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return ""
	}

	return path[:idx]
}

// JoinPath joins a parent path and a leaf field name.
func JoinPath(path string, leaf string) string {
	if path == "" {
		return leaf
	}

	return path + "." + leaf
}
