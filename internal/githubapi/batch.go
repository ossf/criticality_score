// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githubapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hasura/go-graphql-client"
)

func constructBatchQuery[T any](queries map[string]string) (string, map[string]string, error) {
	var t T

	// Serializes type T into GraphQL format. The result is used to build up the
	// batch query.
	queryObj, err := graphql.ConstructQuery(t, map[string]any{})
	if err != nil {
		return "", nil, err
	}
	fieldMap := make(map[string]string)
	query := ""
	idx := 0
	for key, subquery := range queries {
		// Generate a field name to track which result belongs to which query.
		name := fmt.Sprintf("field%d", idx)
		fieldMap[key] = name

		// Join the subquery to the current set of queries, combining it with
		// the field name, and the queryObj generated for T.
		// The value being appended looks like: 'field0:search(foo="bah"){blah}'
		query = query + fmt.Sprintf("%s:%s%s", name, subquery, queryObj)
		idx++
	}
	return "{" + query + "}", fieldMap, nil
}

// BatchQuery can be used to batch a set of requests together to GitHub's
// GraphQL API.
//
// The queries must be for objects represented by type T. T should be a struct
// to work correctly.
func BatchQuery[T any](ctx context.Context, c *Client, queries map[string]string) (map[string]T, error) {
	// TODO: an upper bound should be added
	if len(queries) == 0 {
		// TODO: consider just returning an empty result set rather than panicing.
		panic("no query to run")
	}

	// Generate the query from the type T and the set of queries.
	query, fieldMap, err := constructBatchQuery[T](queries)
	if err != nil {
		return nil, fmt.Errorf("failed to construct batch query: %w", err)
	}

	// Execute the raw query.
	data, err := c.GraphQL().ExecRaw(ctx, query, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("failed executing raw query '%s': %w", query, err)
	}

	// Parse the JSON response into a map where the keys are the field names
	// and the values the results we looked up as type T.
	var resultByField map[string]T
	if err := json.Unmarshal(data, &resultByField); err != nil {
		return nil, fmt.Errorf("json parsing failed: %w", err)
	}

	// Remap the results so the keys supplied in the queries argument are mapped
	// to their results.
	res := map[string]T{}
	for key, name := range fieldMap {
		res[key] = resultByField[name]
	}
	return res, nil
}
