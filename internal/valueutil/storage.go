package valueutil

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseStorageGiB parses a storage string such as "50Gi" into its integer GiB
// value. It only accepts whole-gibibyte ("Gi") values; any other unit or format
// is rejected with an error, since silently misinterpreting it could lead to an
// incorrect disk size being recorded.
func ParseStorageGiB(storage string) (int64, error) {
	trimmed, ok := strings.CutSuffix(strings.TrimSpace(storage), "Gi")
	if !ok {
		return 0, fmt.Errorf("expected a gibibyte value with a \"Gi\" suffix, but got %q", storage)
	}

	value, err := strconv.ParseInt(strings.TrimSpace(trimmed), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse storage value %q as a whole number of GiB: %w", storage, err)
	}

	return value, nil
}
