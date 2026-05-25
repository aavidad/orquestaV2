package orquestadomainworksql

import (
	"context"
	"strconv"
)

func (store *SQLDomainWorkJobRecordStoreV0) placeholderV0(index int) string {
	switch store.placeholderStyle {
	case SQLDomainWorkPlaceholderDollarV0:
		return "$" + strconv.Itoa(index)
	default:
		return "?"
	}
}

func validDomainWorkSQLPlaceholderStyleV0(style SQLDomainWorkPlaceholderStyleV0) bool {
	switch style {
	case SQLDomainWorkPlaceholderQuestionV0, SQLDomainWorkPlaceholderDollarV0:
		return true
	default:
		return false
	}
}

func validDomainWorkSQLIdentifierV0(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || index > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func domainWorkSQLContextV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
