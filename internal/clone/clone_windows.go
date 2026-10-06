package clone

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"dedup/internal/fsutil"
	"golang.org/x/sys/windows"
)

const chunk = 1 << 30

type duplicateExtentsData struct {
	FileHandle       windows.Handle
	_                [8 - unsafe.Sizeof(windows.Handle(0))]byte // match C's 8-byte LARGE_INTEGER alignment on 386
	SourceFileOffset int64
	TargetFileOffset int64
	ByteCount        int64
}

type integrityInfo struct {
	ChecksumAlgorithm        uint16
	Reserved                 uint16
	Flags                    uint32
	ChecksumChunkSizeInBytes uint32
	ClusterSizeInBytes       uint32
}

// Share block-clones src into a new file via FSCTL_DUPLICATE_EXTENTS_TO_FILE (ReFS) and swaps it in for dst.
func Share(src, dst string, size int64) error {
	info, err := os.Stat(dst)
	if err != nil {
		return err
	}
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	cluster, err := clusterSize(s)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrUnsupported, err)
	}

	return fsutil.Replace(dst, func(tmp string) error {
		t, err := os.Create(tmp)
		if err != nil {
			return err
		}
		defer t.Close()
		if err := t.Truncate(size); err != nil {
			return err
		}
		total := (size + cluster - 1) / cluster * cluster
		for off := int64(0); off < total; off += chunk {
			req := duplicateExtentsData{
				FileHandle:       windows.Handle(s.Fd()),
				SourceFileOffset: off,
				TargetFileOffset: off,
				ByteCount:        min(total-off, chunk),
			}
			var n uint32
			err := windows.DeviceIoControl(windows.Handle(t.Fd()), windows.FSCTL_DUPLICATE_EXTENTS_TO_FILE,
				(*byte)(unsafe.Pointer(&req)), uint32(unsafe.Sizeof(req)), nil, 0, &n, nil)
			if err != nil {
				return fmt.Errorf("%w: %v", errors.ErrUnsupported, err)
			}
		}
		if err := t.Close(); err != nil {
			return err
		}
		return fsutil.CopyMeta(info, tmp)
	})
}

func clusterSize(f *os.File) (int64, error) {
	var info integrityInfo
	var n uint32
	err := windows.DeviceIoControl(windows.Handle(f.Fd()), windows.FSCTL_GET_INTEGRITY_INFORMATION,
		nil, 0, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), &n, nil)
	if err != nil {
		return 0, err
	}
	return int64(info.ClusterSizeInBytes), nil
}
