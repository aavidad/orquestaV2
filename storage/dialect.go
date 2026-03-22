package storage

import (
	"strconv"
	"strings"
)

type Dialect struct {
	Name string
}

func DialectForDriver(driver string) Dialect {
	return Dialect{Name: normalizeDriver(driver)}
}

func (d Dialect) Rebind(query string) string {
	if d.Name != "postgres" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	index := 1
	inSingle := false
	inDouble := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		switch ch {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(query) && query[i+1] == '\'' {
					b.WriteByte(ch)
					b.WriteByte(query[i+1])
					i++
					continue
				}
				inSingle = !inSingle
			}
			b.WriteByte(ch)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			b.WriteByte(ch)
		case '?':
			if inSingle || inDouble {
				b.WriteByte(ch)
				continue
			}
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(index))
			index++
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func RebindQuery(driver, query string) string {
	return DialectForDriver(driver).Rebind(query)
}

func (d Dialect) PlaceholderStyle() string {
	if d.Name == "postgres" {
		return "numbered"
	}
	return "qmark"
}

func (d Dialect) RebindParameters() bool {
	return d.Name == "postgres"
}

func (d Dialect) SupportsSchemaBootstrap() bool {
	return d.Name == "sqlite"
}
