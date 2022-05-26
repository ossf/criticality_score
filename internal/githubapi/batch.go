package githubapi

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// BatchQuery can be used to batch a set of requests together to GitHub's
// GraphQL API.
func BatchQuery[T any](ctx context.Context, c *Client, queries map[string]string, vars map[string]any) (map[string]T, error) {
	// Create a query using reflection (see https://github.com/shurcooL/githubv4/issues/17)
	// for when we don't know the exact query before runtime.
	var t T
	fieldType := reflect.TypeOf(t)
	var fields []reflect.StructField
	fieldToKey := map[string]string{}
	idx := 0
	for key, q := range queries {
		name := fmt.Sprintf("Field%d", idx)
		fieldToKey[name] = key
		fields = append(fields, reflect.StructField{
			Name: name,
			Type: fieldType,
			Tag:  reflect.StructTag(fmt.Sprintf(`graphql:"field%d:%s"`, idx, strings.ReplaceAll(q, "\"", "\\\""))),
		})
		idx++
	}
	// TODO: an upper bound should be added
	if len(fields) == 0 {
		// TODO: consider just returning an empty result set rather than panicing.
		panic("no query to run")
	}
	q := reflect.New(reflect.StructOf(fields)).Elem()
	if err := c.GraphQL().Query(ctx, q.Addr().Interface(), vars); err != nil {
		return nil, err
	}
	res := map[string]T{}
	for _, sf := range reflect.VisibleFields(q.Type()) {
		key := fieldToKey[sf.Name]
		v := q.FieldByIndex(sf.Index)
		res[key] = v.Interface().(T)
	}
	return res, nil
}
