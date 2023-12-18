package providertesting

import (
	"github.com/google/uuid"
)

func MatchUUID(value string) error {
	_, err := uuid.Parse(value)
	return err
}
