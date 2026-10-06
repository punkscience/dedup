// Package clone makes dst share src's storage on copy-on-write filesystems, keeping dst's path.
package clone

import "errors"

var ErrDiffers = errors.New("contents differ")
