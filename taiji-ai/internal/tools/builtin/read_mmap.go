package builtin

import (
	"os"

	"golang.org/x/sys/unix"
)

const mmapReadThreshold = 512 * 1024

func readFileBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < mmapReadThreshold {
		return os.ReadFile(path)
	}
	size := int(info.Size())
	if size == 0 {
		return []byte{}, nil
	}
	data, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return os.ReadFile(path)
	}
	defer unix.Munmap(data)
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
