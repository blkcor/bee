package dialect

import "reflect"

var dialectsMap = map[string]Dialect{}

type Dialect interface {
	// DataTypeOf convert the go data type to the db data type
	DataTypeOf(typ reflect.Value) string
	// TableExistSQL return if a table exist sql
	TableExistSQL(tableName string) (string, []interface{})
}

// RegisterDialect register a dialect to the map
func RegisterDialect(name string, dialect Dialect) {
	dialectsMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectsMap[name]
	return
}
