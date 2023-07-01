package tmpl

import (
	"fmt"
	"reflect"
)

type FieldNode struct {
	Value       reflect.Value
	StructField reflect.StructField

	Depth    int
	Parent   *FieldNode
	Children []*FieldNode
}

func (node *FieldNode) IsKind(kind reflect.Kind) (reflect.Kind, bool) {
	if node.StructField.Type.Kind() == reflect.Interface && node.Value.Kind() != kind {
		return node.Value.Kind(), false
	} else if node.StructField.Type.Kind() != kind {
		return node.StructField.Type.Kind(), false
	} else {
		return kind, true
	}
}

func (node *FieldNode) FindPath(path []string) *FieldNode {
	if len(path) == 0 {
		return node
	}

	for _, child := range node.Children {
		if child.StructField.Name == path[0] {
			return child.FindPath(path[1:])
		}
	}

	return nil
}

// createFieldTree can be used to create a tree structure of the fields in a struct
func createFieldTree(structOrPtr interface{}) (root *FieldNode, err error) {
	val := reflect.ValueOf(structOrPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	root = &FieldNode{
		Value: val,
		StructField: reflect.StructField{
			Name: fmt.Sprintf("%T", structOrPtr),
		},

		Depth:    0,
		Parent:   nil,
		Children: make([]*FieldNode, 0),
	}

	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {

		iface := zeroValueInterfaceFromField(val.Field(i))
		if iface != nil {
			node, err := createFieldTree(iface)
			if err != nil {
				return nil, err
			}
			node.StructField = val.Type().Field(i)
			node.Parent = root
			node.Depth = root.Depth + 1
			root.Children = append(root.Children, node)

			//support embedded struct fields
			if node.StructField.Anonymous {
				for _, child := range node.Children {
					child.Parent = root
					child.Depth = root.Depth + 1
					root.Children = append(root.Children, child)
				}
			}
		} else if val.Field(i).Kind() == reflect.Struct {
			node := &FieldNode{
				Value:       val.Field(i),
				StructField: val.Type().Field(i),
				Depth:       root.Depth + 1,
				Parent:      root,
				Children:    make([]*FieldNode, 0),
			}
			root.Children = append(root.Children, node)
		} else {
			node := &FieldNode{
				Value: val.Field(i),
				Depth: root.Depth + 1,
				StructField: reflect.StructField{
					Name: val.Type().Field(i).Name,
					Type: val.Type().Field(i).Type,
				},
				Parent:   root,
				Children: make([]*FieldNode, 0),
			}
			root.Children = append(root.Children, node)
		}
	}

	return root, nil
}

func recurseFieldsImplementing[T interface{}](structOrPtr interface{}, fn func(val T, field reflect.StructField) error) error {
	val := reflect.ValueOf(structOrPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	iface := zeroValueInterfaceFromField(val)
	if t, ok := iface.(T); ok {
		err := fn(t, reflect.StructField{
			Name: fmt.Sprintf("%T", structOrPtr),
		})
		if err != nil {
			return err
		}
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() != reflect.Ptr &&
			field.Kind() != reflect.Slice &&
			field.Kind() != reflect.Struct {
			continue
		}

		iface := zeroValueInterfaceFromField(field)
		if t, ok := iface.(T); ok {
			err := fn(t, val.Type().Field(i))
			if err != nil {
				return err
			}

			err = recurseFieldsImplementing[T](t, fn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// zeroValueInterfaceFromField converts a reflected field to a zero'd version of itself as an interface type.
// this makes it easier to perform type assertions on reflected struct fields
func zeroValueInterfaceFromField(field reflect.Value) interface{} {
	switch field.Kind() {
	case reflect.Struct:
		if field.Type().Kind() == reflect.Ptr {
			return reflect.New(field.Type().Elem()).Interface()
		} else {
			return reflect.New(field.Type()).Interface()
		}
	case reflect.Ptr:
		fallthrough
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Ptr {
			return reflect.New(field.Type().Elem().Elem()).Interface()
		} else {
			return reflect.New(field.Type().Elem()).Interface()
		}
	}
	return nil
}
