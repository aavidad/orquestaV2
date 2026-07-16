//go:build !unix

package sqlite

import "errors"

func (c *conn) FileControlFileDescriptor(string) (int, error) {
	return -1, errors.New("sqlite: file descriptor unsupported")
}
