package sqlite

import (
	"context"
	"database/sql/driver"
	"errors"
	"os"

	moderncsqlite "modernc.org/sqlite"
)

type localStateConnectionIdentity struct {
	device uint64
	inode  uint64
}

type localStateConnector struct {
	base     driver.Connector
	validate func(moderncsqlite.FileControl) error
}

func newLocalStateConnector(dsn string, witness *os.File, supported bool) (driver.Connector, error) {
	connector := &localStateConnector{}
	if !supported {
		connector.base = moderncsqlite.NewConnector(dsn)
		return connector, nil
	}
	if witness == nil {
		return nil, errors.New("sqlite.local_state_connection_identity_unavailable")
	}
	info, err := witness.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("sqlite.local_state_connection_identity_unavailable")
	}
	device, inode, reliable := localFileIdentity(info)
	if !reliable {
		return nil, errors.New("sqlite.local_state_connection_identity_unavailable")
	}
	expected := localStateConnectionIdentity{device: device, inode: inode}
	connector.validate = func(control moderncsqlite.FileControl) error {
		return validateLocalStateConnection(control, expected)
	}
	connector.base = moderncsqlite.NewValidatedConnector(dsn, func(control moderncsqlite.FileControl) error {
		return connector.validate(control)
	})
	return connector, nil
}

func (connector *localStateConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return connector.base.Connect(ctx)
}

func (connector *localStateConnector) Driver() driver.Driver {
	return connector.base.Driver()
}
