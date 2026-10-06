//go:build !unix && !windows

package fileid

import "io/fs"

func Of(string, fs.FileInfo) (ID, error) {
	return ID{}, nil
}
