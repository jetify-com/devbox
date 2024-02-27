package configfile

import (
	"reflect"
	"strings"

	"github.com/tailscale/hujson"
)

func (c *ConfigFile) SetStringField(fieldName, val string) {
	valueOfStruct := reflect.ValueOf(c).Elem()

	field := valueOfStruct.FieldByName(fieldName)
	field.SetString(val)

	c.ast.setStringField(c.jsonNameOfField(fieldName), val)
}

func (c *ConfigFile) jsonNameOfField(fieldName string) string {
	valueOfStruct := reflect.ValueOf(c).Elem()

	var name string
	for i := 0; i < valueOfStruct.NumField(); i++ {
		field := valueOfStruct.Type().Field(i)
		if field.Name != fieldName {
			continue
		}

		name = field.Name
		jsonTag := field.Tag.Get("json")
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			name = parts[0]
		}

		break
	}
	return name
}

func (c *configAST) setStringField(key, val string) {
	rootObject := c.root.Value.(*hujson.Object)
	i := c.memberIndex(rootObject, key)
	if i == -1 {
		rootObject.Members = append(rootObject.Members, hujson.ObjectMember{
			Name:  hujson.Value{Value: hujson.String(key)},
			Value: hujson.Value{Value: hujson.String(val)},
		})
	} else if val != "" {
		rootObject.Members[i].Value = hujson.Value{Value: hujson.String(val)}
	} else {
		rootObject.Members = append(rootObject.Members[:i], rootObject.Members[i+1:]...)
	}

	c.root.Format()
}
