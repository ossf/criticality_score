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

package depsdev

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

var ErrorNoResults = errors.New("no results returned")

type Dataset struct {
	ds *bigquery.Dataset
}

type Table struct {
	md *bigquery.TableMetadata
}

// bqAPI wraps the BigQuery Go API to make the deps.dev implementation easier to unit test.
type bqAPI interface {
	Project() string
	OneResultQuery(ctx context.Context, query string, params map[string]any, result any) error
	NoResultQuery(ctx context.Context, query string, params map[string]any) error
	GetDataset(ctx context.Context, id string) (*Dataset, error)
	CreateDataset(ctx context.Context, id string, ttl time.Duration) (*Dataset, error)
	UpdateDataset(ctx context.Context, d *Dataset, ttl time.Duration) error
	GetTable(ctx context.Context, d *Dataset, id string) (*Table, error)
}

type bq struct {
	client *bigquery.Client
}

func (b *bq) Project() string {
	return b.client.Project()
}

func (b *bq) OneResultQuery(ctx context.Context, query string, params map[string]any, result any) error {
	q := b.client.Query(query)
	for k, v := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{Name: k, Value: v})
	}
	it, err := q.Read(ctx)
	if err != nil {
		return err
	}
	err = it.Next(result)
	if errors.Is(err, iterator.Done) {
		return ErrorNoResults
	}
	if err != nil {
		return err
	}
	return nil
}

func (b *bq) NoResultQuery(ctx context.Context, query string, params map[string]any) error {
	q := b.client.Query(query)
	for k, v := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{Name: k, Value: v})
	}
	j, err := q.Run(ctx)
	if err != nil {
		return err
	}
	status, err := j.Wait(ctx)
	if err != nil {
		return err
	}
	if err := status.Err(); err != nil {
		return err
	}
	return nil
}

func (b *bq) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	ds := b.client.Dataset(id)
	_, err := ds.Metadata(ctx)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dataset metadata: %w", err)
	}
	return &Dataset{
		ds: ds,
	}, nil
}

func (b *bq) CreateDataset(ctx context.Context, id string, ttl time.Duration) (*Dataset, error) {
	ds := b.client.Dataset(id)
	if err := ds.Create(ctx, &bigquery.DatasetMetadata{DefaultTableExpiration: ttl}); err != nil {
		return nil, err
	}
	return &Dataset{
		ds: ds,
	}, nil
}

func (b *bq) UpdateDataset(ctx context.Context, d *Dataset, ttl time.Duration) error {
	md, err := d.ds.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("dataset metadata: %w", err)
	}
	var update bigquery.DatasetMetadataToUpdate
	needsWrite := false
	if md.DefaultTableExpiration != ttl {
		update.DefaultTableExpiration = ttl
		needsWrite = true
	}
	if needsWrite {
		if _, err := d.ds.Update(ctx, update, ""); err != nil {
			return fmt.Errorf("dataset update metadata: %w", err)
		}
	}
	return nil
}

func (b *bq) GetTable(ctx context.Context, d *Dataset, id string) (*Table, error) {
	t := d.ds.Table(id)
	md, err := t.Metadata(ctx)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Table{md: md}, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *googleapi.Error
	ok := errors.As(err, &apiErr)
	return ok && apiErr.Code == 404
}
