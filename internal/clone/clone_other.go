//go:build !linux && !darwin && !windows

package clone

import "errors"

func Share(string, string, int64) error {
	return errors.ErrUnsupported
}
