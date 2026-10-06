package clone

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const chunk = 16 << 20

// Share uses FIDEDUPERANGE, so the kernel verifies the bytes match before sharing them.
func Share(src, dst string, size int64) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(dst, os.O_RDWR, 0)
	if errors.Is(err, fs.ErrPermission) {
		d, err = os.Open(dst)
	}
	if err != nil {
		return err
	}
	defer d.Close()

	for off := uint64(0); off < uint64(size); {
		r := unix.FileDedupeRange{
			Src_offset: off,
			Src_length: min(uint64(size)-off, chunk),
			Info:       []unix.FileDedupeRangeInfo{{Dest_fd: int64(d.Fd()), Dest_offset: off}},
		}
		if err := unix.IoctlFileDedupeRange(int(s.Fd()), &r); err != nil {
			return classify(err)
		}
		info := r.Info[0]
		switch {
		case info.Status == unix.FILE_DEDUPE_RANGE_DIFFERS:
			return ErrDiffers
		case info.Status < 0:
			return classify(syscall.Errno(-info.Status))
		case info.Bytes_deduped == 0:
			return fmt.Errorf("no progress at offset %d", off)
		}
		off += info.Bytes_deduped
	}
	return nil
}

func classify(err error) error {
	switch {
	case errors.Is(err, unix.EOPNOTSUPP), errors.Is(err, unix.EINVAL),
		errors.Is(err, unix.EXDEV), errors.Is(err, unix.ENOTTY):
		return fmt.Errorf("%w: %v", errors.ErrUnsupported, err)
	}
	return err
}
