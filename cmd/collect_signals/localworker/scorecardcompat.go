// Copyright 2023 Criticality Score Authors
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

package localworker

import (
	"context"
	"fmt"
	"time"

	"github.com/ossf/scorecard/v4/clients"
	"github.com/ossf/scorecard/v4/cron/data"
	"github.com/ossf/scorecard/v4/cron/worker"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ossf/criticality_score/v2/cmd/collect_signals/vcs"
)

var headSHA = clients.HeadSHA

func writeMetadata(ctx context.Context, bucketURL string, lastShard int32, t time.Time) error {
	// Populate `.shard_metadata` file.
	metadata := data.ShardMetadata{
		NumShard:  new(int32),
		ShardLoc:  new(string),
		CommitSha: new(string),
	}
	*metadata.NumShard = lastShard + 1
	*metadata.ShardLoc = bucketURL + "/" + data.GetBlobFilename("", t)
	*metadata.CommitSha = vcs.CommitID()

	metadataJSON, err := protojson.Marshal(&metadata)
	if err != nil {
		return fmt.Errorf("protojson marshal: %w", err)
	}

	if err := data.WriteToBlobStore(ctx, bucketURL, data.GetShardMetadataFilename(t), metadataJSON); err != nil {
		return fmt.Errorf("writing to blob store: %w", err)
	}
	return nil
}

func makeRequest(repos []string, shardNum int32, t time.Time) *data.ScorecardBatchRequest {
	req := &data.ScorecardBatchRequest{
		JobTime:  timestamppb.New(t),
		ShardNum: proto.Int32(shardNum),
	}
	for _, repo := range repos {
		localRepo := repo
		req.Repos = append(req.GetRepos(), &data.Repo{
			Url:      proto.String(localRepo),
			Commit:   proto.String(headSHA),
			Metadata: []string{},
		})
	}
	return req
}

func resultExists(ctx context.Context, sbr *data.ScorecardBatchRequest, bucketURL string) (bool, error) {
	exists, err := data.BlobExists(ctx, bucketURL, worker.ResultFilename(sbr))
	if err != nil {
		return false, fmt.Errorf("blob exists check: %w", err)
	}
	return exists, nil
}
