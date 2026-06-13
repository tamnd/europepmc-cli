package cli

import (
	"errors"

	"github.com/tamnd/europepmc-cli/europepmc"
)

func isNotFound(err error) bool {
	return errors.Is(err, europepmc.ErrNotFound)
}
