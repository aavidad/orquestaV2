package orquestaruntimecodexdelivery

import (
	"fmt"
	"io"
	"os"
)

const (
	codexDeliveryFileSnapshotMaxBytesV0   int64 = 16 * 1024 * 1024
	codexDeliveryFileSnapshotMaxRecordsV0       = 10000
)

func readCodexDeliveryFileSnapshotBytesV0(path string, owner string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s: read_failed", owner)
	}
	if info.Size() > codexDeliveryFileSnapshotMaxBytesV0 {
		return nil, fmt.Errorf("%s: size_limit_exceeded", owner)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: read_failed", owner)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, codexDeliveryFileSnapshotMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("%s: read_failed", owner)
	}
	if int64(len(data)) > codexDeliveryFileSnapshotMaxBytesV0 {
		return nil, fmt.Errorf("%s: size_limit_exceeded", owner)
	}
	return data, nil
}
