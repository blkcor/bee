package schema

import (
	"github.com/blkcor/beeORM/dialect"
	"go/ast"
	"reflect"
)

// Field represents a column of database
type Field struct {
	Name string
	Type string
	Tag  string
}

// Schema represents a table of database
type Schema struct {
	Name       string
	Model      interface{}
	Fields     []*Field
	FieldNames []string
	fieldMap   map[string]*Field
}

func (schema *Schema) GetField(fieldName string) *Field {
	return schema.fieldMap[fieldName]
}

// Parse the object to a schema instance
func Parse(dest interface{}, dialect dialect.Dialect) *Schema {
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Name:     modelType.Name(),
		Model:    dest,
		fieldMap: make(map[string]*Field),
	}

	for i := 0; i < modelType.NumField(); i++ {
		attr := modelType.Field(i)
		if !attr.Anonymous && ast.IsExported(attr.Name) {
			field := &Field{
				Name: attr.Name,
				Type: dialect.DataTypeOf(reflect.Indirect(reflect.New(attr.Type))),
			}
			if v, ok := attr.Tag.Lookup("beeorm"); ok {
				field.Tag = v
			}
			schema.FieldNames = append(schema.FieldNames, field.Name)
			schema.Fields = append(schema.Fields, field)
			schema.fieldMap[field.Name] = field
		}
	}
	return schema
}
