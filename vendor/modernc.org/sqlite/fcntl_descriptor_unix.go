//go:build unix

package sqlite

import (
	"errors"
	"unsafe"

	sqlite3 "modernc.org/sqlite/lib"
)

// FileControlFileDescriptor exposes SQLite's borrowed main-file descriptor so
// callers inside database/sql.Conn.Raw can bind security decisions to the file
// that the active connection actually opened.
func (c *conn) FileControlFileDescriptor(dbName string) (int, error) {
	pointerSlot := c.tls.Alloc(int(unsafe.Sizeof(uintptr(0))))
	defer c.tls.Free(int(unsafe.Sizeof(uintptr(0))))

	*(*uintptr)(unsafe.Pointer(pointerSlot)) = 0
	if err := c.fileControl(dbName, sqlite3.SQLITE_FCNTL_FILE_POINTER, pointerSlot); err != nil {
		return -1, err
	}
	filePointer := *(*uintptr)(unsafe.Pointer(pointerSlot))
	if filePointer == 0 {
		return -1, errors.New("sqlite: file descriptor unavailable")
	}
	descriptor := (*sqlite3.TunixFile)(unsafe.Pointer(filePointer)).Fh
	if descriptor < 0 {
		return -1, errors.New("sqlite: file descriptor unavailable")
	}
	return int(descriptor), nil
}
