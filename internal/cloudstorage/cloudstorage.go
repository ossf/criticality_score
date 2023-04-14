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

package cloudstorage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)

const fileScheme = "file"

func parseBucketAndPrefix(rawURL string) (bucket, prefix string, _ error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("url parse: %w", err)
	}

	// If the URL doesn't have a scheme it is possibly a local file. Use the
	// fileblob storage to handle these files because the behavior is
	// more consistent with cloud storage services - in particular
	// atomic updates, and read-after-write consistency.
	if !u.IsAbs() {
		// If the Host is set (e.g. //example.com/) then we have a problem.
		if u.Host != "" {
			return "", "", fmt.Errorf("undefined blob scheme: %s", u.String())
		}

		// Assume a scheme-less, host-less url is a local file.
		u.Scheme = fileScheme
		if !path.IsAbs(u.Path) {
			u.Host = "."
		}
		// Turn off .attrs files, becaue they look weird next to local files.
		q := u.Query()
		q.Set("metadata", "skip")
		u.RawQuery = q.Encode()
	}

	if u.Scheme == fileScheme {
		// File schemes are treated differently, as the dir forms the bucket.
		u.Path, prefix = path.Split(u.Path)
	} else {
		prefix = strings.TrimPrefix(u.Path, "/")
		u.Path = ""
	}

	bucket = u.String()
	return bucket, prefix, nil
}

func NewWriter(ctx context.Context, rawURL string) (io.WriteCloser, error) {
	bucket, prefix, err := parseBucketAndPrefix(rawURL)
	if err != nil {
		return nil, err
	}

	b, err := blob.OpenBucket(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed opening %s: %w", bucket, err)
	}
	w, err := b.NewWriter(ctx, prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating writer for %s: %w", rawURL, err)
	}
	return w, nil
}
