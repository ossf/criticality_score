package depsdev

import (
	"bytes"
	"context"
	"errors"
	"text/template"
	"time"

	"cloud.google.com/go/bigquery"
	log "github.com/sirupsen/logrus"
	_ "google.golang.org/api/bigquery/v2"
)

const (
	dependentCountsTableName         = "dependent_counts"
	packageVersionToProjectTableName = "package_version_to_project"

	snapshotQuery = "SELECT MAX(Time) AS SnapshotTime FROM `bigquery-public-data.deps_dev_v1.Snapshots`"
)

// TODO: prune root dependents that come from the same project.
// TODO: count "# packages per project" to determine dependent ratio

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

CREATE TABLE ` + "`{{.ProjectID}}.{{.DatasetName}}.{{.TableName}}`" + `
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

func NewDependents(ctx context.Context, client *bigquery.Client, logger *log.Logger, datasetName string) (*dependents, error) {
	b := &bq{client: client}
	c := &dependents{
		b: b,
		logger: logger.WithFields(log.Fields{
			"project_id": b.Project(),
			"dataset":    datasetName,
		}),
		datasetName: datasetName,
	}
	var err error

	// Populate the snapshot time
	c.snapshotTime, err = c.getLatestSnapshotTime(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure the dataset exists
	ds, err := c.getOrCreateDataset(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure the dependent count table exists and is populated
	t, err := c.b.GetTable(ctx, ds, dependentCountsTableName)
	if err != nil {
		return nil, err
	}
	if t != nil {
		c.logger.Warn("dependent count table exists")
	} else {
		c.logger.Warn("creating dependent count table")
		err := c.b.NoResultQuery(ctx, c.generateQuery(dataQuery), map[string]any{"part": c.snapshotTime})
		if err != nil {
			return nil, err
		}
	}

	// Cache the data query to avoid re-generating it repeatedly.
	c.countQuery = c.generateQuery(countQuery)

	return c, nil
}

type dependents struct {
	b            bqAPI
	logger       *log.Entry
	snapshotTime time.Time
	countQuery   string
	datasetName  string
}

func (c *dependents) generateQuery(temp string) string {
	t := template.Must(template.New("query").Parse(temp))
	var b bytes.Buffer
	t.Execute(&b, struct {
		ProjectID   string
		DatasetName string
		TableName   string
	}{c.b.Project(), c.datasetName, dependentCountsTableName})
	return b.String()
}

func (c *dependents) Count(ctx context.Context, projectName, projectType string) (int, bool, error) {
	var rec struct {
		DependentCount int
	}
	params := map[string]any{
		"projectname": projectName,
		"projecttype": projectType,
	}
	err := c.b.OneResultQuery(ctx, c.countQuery, params, &rec)
	if err == nil {
		return rec.DependentCount, true, nil
	}
	if errors.Is(err, NoResultError) {
		return 0, false, nil
	}
	return 0, false, err
}

func (c *dependents) LatestSnapshotTime() time.Time {
	return c.snapshotTime
}

func (c *dependents) getLatestSnapshotTime(ctx context.Context) (time.Time, error) {
	var rec struct {
		SnapshotTime time.Time
	}
	if err := c.b.OneResultQuery(ctx, snapshotQuery, nil, &rec); err != nil {
		return time.Time{}, err
	}
	return rec.SnapshotTime, nil
}

func (c *dependents) getOrCreateDataset(ctx context.Context) (*Dataset, error) {
	// Attempt to get the current dataset
	ds, err := c.b.GetDataset(ctx, c.datasetName)
	if err != nil {
		return nil, err
	}
	if ds == nil {
		// Dataset doesn't exist so create it
		c.logger.Debug("creating dependent count dataset")
		ds, err = c.b.CreateDataset(ctx, c.datasetName)
		if err != nil {
			return nil, err
		}
	} else {
		c.logger.Debug("dependent count dataset exists")
	}
	return ds, nil
}
