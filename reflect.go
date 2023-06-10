package tmpl

import (
	"fmt"
	"reflect"
)

type fieldNode struct {
	reflect.StructField

	Depth    int
	Parent   *fieldNode
	Children []*fieldNode
}

// createFieldTree can be used to create a tree structure of the fields in a struct
// that implement the given generic interface type
func createFieldTree(structOrPtr interface{}) (root *fieldNode, err error) {
	root = &fieldNode{
		StructField: reflect.StructField{
			Name: fmt.Sprintf("%T", structOrPtr),
		},

		Depth:    0,
		Parent:   nil,
		Children: make([]*fieldNode, 0),
	}

	val := reflect.ValueOf(structOrPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
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
		} else {
			node := &fieldNode{
				StructField: val.Type().Field(i),
				Depth:       root.Depth + 1,
				Parent:      root,
				Children:    make([]*fieldNode, 0),
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

			err = recurseFieldsImplementing[T](reflect.ValueOf(t).Elem(), fn)
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
