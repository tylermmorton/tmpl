package tmpl

import (
	"fmt"
	"reflect"
)

func recurseFieldsImplementing[T interface{}](structOrPtr interface{}, fn func(val T, field reflect.StructField) error) error {
	val := reflect.ValueOf(structOrPtr)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	iface := zeroInterfaceFromField(val)
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

		iface := zeroInterfaceFromField(field)
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

// zeroInterfaceFromField converts a reflected field to a zero'd version of itself as an interface type.
// this makes it easier to perform type assertions on reflected struct fields
func zeroInterfaceFromField(field reflect.Value) interface{} {
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
