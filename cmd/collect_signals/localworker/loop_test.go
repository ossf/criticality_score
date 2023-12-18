package localworker

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/ossf/scorecard/v4/cron/data"
	"go.uber.org/zap/zaptest"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"

	"github.com/ossf/criticality_score/internal/iterator"
)

type testWorker struct {
	LastReq    *data.ScorecardBatchRequest
	RepoCount  int
	BatchCount int
}

func (w *testWorker) Process(ctx context.Context, req *data.ScorecardBatchRequest, url string) error {
	w.BatchCount += 1
	w.RepoCount += len(req.GetRepos())
	w.LastReq = req
	return nil
}

func (w *testWorker) PostProcess() {}

// TestHappyPath is a functional test for WorkLoop to cover basic happy path
// behaviour. The test covers batch iteration, writing shard metadata,
// and the state file.
func TestHappyPath(t *testing.T) {
	overrideScorecardConfig(t)

	wl, w := makeTestWorkLoop(t, 0)

	if err := wl.Run(); err != nil {
		t.Fatalf("Run() = %v; want no error", err)
	}
	if w.BatchCount != 3 {
		t.Errorf("w.BatchCount = %d; want 3", w.BatchCount)
	}
	if w.RepoCount != 5 {
		t.Errorf("w.RepoCount = %d; want 5", w.RepoCount)
	}

	dataBucket, err := blob.OpenBucket(context.Background(), wl.bucketURL)
	if err != nil {
		t.Fatalf("OpenBucket#1 = %v; want no error", err)
	}
	objs, _, err := dataBucket.ListPage(context.Background(), blob.FirstPageToken, 100, &blob.ListOptions{
		Delimiter: "",
	})
	if err != nil {
		t.Fatalf("ListPage#1 = %v; want no error", err)
	}

	wantKeys := []string{"2023.12.25/123456/.shard_metadata"}
	var gotKeys []string
	for _, obj := range objs {
		gotKeys = append(gotKeys, obj.Key)
	}
	if !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("Data bucket = %v; want %v", gotKeys, wantKeys)
	}

	rawDataBucket, err := blob.OpenBucket(context.Background(), wl.rawBucketURL)
	if err != nil {
		t.Fatalf("OpenBucket#2 = %v; want no error", err)
	}
	rawObjs, _, err := rawDataBucket.ListPage(context.Background(), blob.FirstPageToken, 100, &blob.ListOptions{
		Delimiter: "",
	})
	if err != nil {
		t.Fatalf("ListPage#2 = %v; want no error", err)
	}
	var gotRawKeys []string
	for _, obj := range rawObjs {
		gotRawKeys = append(gotRawKeys, obj.Key)
	}
	if !slices.Equal(gotRawKeys, wantKeys) {
		t.Errorf("Raw data bucket = %v; want %v", gotRawKeys, wantKeys)
	}
}

func TestRecovery(t *testing.T) {
	overrideScorecardConfig(t)

	wl, w := makeTestWorkLoop(t, 1)

	if err := wl.Run(); err != nil {
		t.Fatalf("Run() = %v; want no error", err)
	}
	if w.BatchCount != 2 {
		t.Errorf("w.BatchCount = %d; want 3", w.BatchCount)
	}
	if w.RepoCount != 3 {
		t.Errorf("w.RepoCount = %d; want 5", w.RepoCount)
	}
}

func overrideScorecardConfig(t *testing.T) {
	t.Helper()
	// Set important config values to ensure the tests don't fire
	t.Setenv("SCORECARD_PROJECT_ID", "-not-a-package-id-")
	t.Setenv("SCORECARD_DATA_BUCKET_URL", "")
	t.Setenv("SCORECARD_RAW_RESULT_DATA_BUCKET_URL", "")
	t.Setenv("SCORECARD_SHARD_SIZE", "")
}

func makeTestWorkLoop(t *testing.T, startShard int32) (*WorkLoop, *testWorker) {
	t.Helper()

	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "data"), 0o777)
	os.Mkdir(filepath.Join(d, "rawdata"), 0o777)

	logger := zaptest.NewLogger(t)
	w := &testWorker{}
	wl := &WorkLoop{
		logger:        logger,
		w:             w,
		input:         iterator.Slice([]string{"1", "2", "3", "4", "5"}),
		stateFilename: filepath.Join(d, "statefile"),
		bucketURL:     "file://" + filepath.Join(d, "data"),
		rawBucketURL:  "file://" + filepath.Join(d, "rawdata"),
		shardSize:     2,
	}

	state := &runState{
		filename: wl.stateFilename,
		JobTime:  time.Date(2023, 12, 25, 12, 34, 56, 789, time.UTC),
		Shard:    startShard,
	}
	if err := state.Save(); err != nil {
		t.Fatalf("state.Save() = %v; want no error", err)
	}
	return wl, w
}
