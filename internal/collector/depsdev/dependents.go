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
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/template"
	"time"

	"cloud.google.com/go/bigquery"
	"go.uber.org/zap"
	_ "google.golang.org/api/bigquery/v2"
)

const (
	dependentCountsTableName = "dependent_counts"

	snapshotQuery = "SELECT MAX(Time) AS SnapshotTime FROM `bigquery-public-data.deps_dev_v1.Snapshots`"
)

// TODO: prune root dependents that come from the same project.
// TODO: count "# packages per project" to determine dependent ratio

/*
The SQL statement below is two different queries that are used to query data from BigQuery.
The first query, dataQuery, creates a temporary table rawDependentCounts which holds
information about the name, version, system, and dependent count of dependencies.
This table is created by joining two other tables, Dependencies and PackageVersions,
and counting the number of dependencies for each name, version, and system.

The second query, countQuery, selects the DependentCount from a table specified by the ProjectID, DatasetName,
and TableName placeholders. The table is filtered by the ProjectName and ProjectType parameters,
which are passed as @projectname and @projecttype respectively.
*/

const dataQuery = `
CREATE TEMP TABLE rawDependentCounts(Name STRING, Version STRING, System STRING, DependentCount INT)
AS
  SELECT d.Dependency.Name as Name, d.Dependency.Version as Version, d.Dependency.System as System, COUNT(1) AS DependentCount
  FROM ` + "`bigquery-public-data.deps_dev_v1.Dependencies`" + `AS d
  JOIN (SELECT System, Name, Version, ROW_NUMBER() OVER (PARTITION BY Name ORDER BY VersionInfo.Ordinal Desc) AS RowNumber
   FROM ` + "`bigquery-public-data.deps_dev_v1.PackageVersions`" + `
   WHERE SnapshotAt = @part) AS lv ON (lv.RowNumber = 1 AND lv.Name = d.Name AND lv.Version = d.Version AND lv.System = d.System)
  WHERE d.SnapshotAt = @part
  GROUP BY Name, Version, System;

CREATE TABLE IF NOT EXISTS ` + "`{{.ProjectID}}.{{.DatasetName}}.{{.TableName}}`" + `
AS
WITH pvp AS (
    SELECT System, Name, Version, ProjectName, ProjectType
    FROM ` + "`bigquery-public-data.deps_dev_v1.PackageVersionToProject`" + `
    WHERE SnapshotAt = @part
)
SELECT pvp.ProjectName AS ProjectName, pvp.ProjectType AS ProjectType, SUM(d.DependentCount) AS DependentCount
 FROM pvp
 JOIN rawDependentCounts AS d
      ON (pvp.System = d.System AND pvp.Name = d.Name AND pvp.Version = d.Version)
GROUP BY ProjectName, ProjectType;
`

const countQuery = `
SELECT DependentCount
FROM ` + "`{{.ProjectID}}.{{.DatasetName}}.{{.TableName}}`" + `
WHERE ProjectName = @projectname AND ProjectType = @projecttype;
`

func NewDependents(ctx context.Context, client *bigquery.Client, logger *zap.Logger, datasetName string, datasetTTL time.Duration) (*dependents, error) {
	b := &bq{client: client}
	c := &dependents{
		b: b,
		logger: logger.With(
			zap.String("project_id", b.Project()),
			zap.String("dataset", datasetName),
		),
		datasetName: datasetName,
		datasetTTL:  datasetTTL,
	}
	var err error

	// Ensure the dataset exists
	c.ds, err = c.getOrCreateDataset(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

type cache struct {
	countQuery string
	tableKey   string
}

type dependents struct {
	b            bqAPI
	logger       *zap.Logger
	ds           *Dataset
	lastUseCache *cache
	datasetName  string
	datasetTTL   time.Duration
}

func (c *dependents) generateQuery(queryTemplate, tableName string) string {
	t := template.Must(template.New("query").Parse(queryTemplate))
	var b bytes.Buffer
	t.Execute(&b, struct {
		ProjectID   string
		DatasetName string
		TableName   string
	}{c.b.Project(), c.datasetName, tableName})
	return b.String()
}

func (c *dependents) Count(ctx context.Context, projectName, projectType, tableKey string) (int, bool, error) {
	query, err := c.prepareCountQuery(ctx, tableKey)
	if err != nil {
		return 0, false, fmt.Errorf("prepare count query: %w", err)
	}

	var rec struct {
		DependentCount int
	}
	params := map[string]any{
		"projectname": projectName,
		"projecttype": projectType,
	}
	err = c.b.OneResultQuery(ctx, query, params, &rec)
	if err == nil {
		return rec.DependentCount, true, nil
	}
	if errors.Is(err, ErrorNoResults) {
		return 0, false, nil
	}
	return 0, false, fmt.Errorf("count query: %w", err)
}

func (c *dependents) getLatestSnapshotTime(ctx context.Context) (time.Time, error) {
	var rec struct {
		SnapshotTime time.Time
	}
	if err := c.b.OneResultQuery(ctx, snapshotQuery, nil, &rec); err != nil {
		return time.Time{}, fmt.Errorf("snapshot query: %w", err)
	}
	return rec.SnapshotTime, nil
}

func (c *dependents) getOrCreateDataset(ctx context.Context) (*Dataset, error) {
	// Attempt to get the current dataset
	ds, err := c.b.GetDataset(ctx, c.datasetName)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		// Dataset doesn't exist so create it
		c.logger.Debug("creating dependent count dataset")
		ds, err = c.b.CreateDataset(ctx, c.datasetName, c.datasetTTL)
		if err != nil {
			return nil, fmt.Errorf("create dataset: %w", err)
		}
	} else {
		c.logger.Debug("dependent count dataset exists")
	}
	// Ensure the dataset is consistent with the settings.
	err = c.b.UpdateDataset(ctx, ds, c.datasetTTL)
	if err != nil {
		return nil, fmt.Errorf("update dataset: %w", err)
	}
	return ds, nil
}

func (c *dependents) prepareCountQuery(ctx context.Context, tableKey string) (string, error) {
	// If the last use is the same as this usage, avoid needlessly checking if
	// the table exists.
	if c.lastUseCache != nil && c.lastUseCache.tableKey == tableKey {
		c.logger.Debug("Using cached dependent count query")
		return c.lastUseCache.countQuery, nil
	}

	// Generate the table name
	tableName := getTableName(tableKey)

	// Ensure the dependent count table exists and is populated
	t, err := c.b.GetTable(ctx, c.ds, tableName)
	if err != nil {
		return "", fmt.Errorf("get table: %w", err)
	}
	if t != nil {
		c.logger.Info("Dependent count table exists")
	} else {
		c.logger.Info("Creating dependent count table")
		// Always get the latest snapshot time to ensure the partition used is
		// the latest possible partition.
		snapshotTime, err := c.getLatestSnapshotTime(ctx)
		if err != nil {
			return "", fmt.Errorf("get latest snapshot time: %w", err)
		}
		// This query use "IF NOT EXISTS" to avoid failing if two or more
		// workers execute this code at the same time.
		err = c.b.NoResultQuery(ctx, c.generateQuery(dataQuery, tableName), map[string]any{"part": snapshotTime})
		if err != nil {
			return "", fmt.Errorf("create table: %w", err)
		}
	}

	c.lastUseCache = &cache{
		tableKey:   tableKey,
		countQuery: c.generateQuery(countQuery, tableName),
	}
	return c.lastUseCache.countQuery, nil
}

func getTableName(tableKey string) string {
	if tableKey == "" {
		return dependentCountsTableName
	}
	return dependentCountsTableName + "_" + tableKey
}
