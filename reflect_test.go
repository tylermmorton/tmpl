package tmpl

import (
	"strings"
	"testing"
)

type testFieldTree struct {
}

func (*testFieldTree) Method1() error {
	return nil
}

type testReturnType struct {
	Field1 string
}

func (*testFieldTree) Method2() (*testReturnType, error) {
	return &testReturnType{}, nil
}

func (t *testFieldTree) Method3() *testFieldTree {
	return t
}

func (t *testFieldTree) Method4() testFieldTree {
	return *t
}

func Test_createFieldTree(t *testing.T) {
	testTable := []struct {
		name        string
		structOrPtr interface{}
		wantFields  []string
		wantErr     bool
	}{
		{
			name:        "Detects methods attached via pointer receiver",
			structOrPtr: &testFieldTree{},
			wantFields: []string{
				".Method1",
				".Method2.Field1",
				".Method3.Method1",
			},
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			fieldTree, err := createFieldTree(tt.structOrPtr)
			if (err != nil) != tt.wantErr {
				t.Errorf("createFieldTree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, field := range tt.wantFields {
				name := strings.TrimPrefix(field, ".")
				node := fieldTree.FindPath(strings.Split(name, "."))
				if node == nil {
					t.Errorf("createFieldTree() field %q not found", field)
				}
			}
		})
	}
}
