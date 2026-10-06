package clone

import (
	"errors"
	"fmt"
	"os"

	"dedup/internal/fsutil"
	"golang.org/x/sys/unix"
)

func Share(src, dst string, _ int64) error {
	info, err := os.Stat(dst)
	if err != nil {
		return err
	}
	return fsutil.Replace(dst, func(tmp string) error {
		if err := unix.Clonefile(src, tmp, unix.CLONE_NOFOLLOW); err != nil {
			if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EXDEV) {
				return fmt.Errorf("%w: %v", errors.ErrUnsupported, err)
			}
			return err
		}
		return fsutil.CopyMeta(info, tmp)
	})
}
