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

package outfile

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)

// fileOpener wraps a method for opening files.
//
// This allows tests to fake the behavior of os.OpenFile() to avoid hitting
// the filesystem.
type fileOpener interface {
	Open(string, int, os.FileMode) (*os.File, error)
}

// fileOpenerFunc allows a function to implement the openFileWrapper interface.
//
// This is convenient for wrapping os.OpenFile().
type fileOpenerFunc func(string, int, os.FileMode) (*os.File, error)

func (f fileOpenerFunc) Open(filename string, flags int, perm os.FileMode) (*os.File, error) {
	return f(filename, flags, perm)
}

type Opener struct {
	fileOpener fileOpener
	StdoutName string
	forceFlag  string
	force      bool
	append     bool
	Perm       os.FileMode
}

// CreateOpener creates an Opener and defines the sepecified flags forceFlag and appendFlag.
func CreateOpener(fs *flag.FlagSet, forceFlag, appendFlag, fileHelpName string) *Opener {
	o := &Opener{
		Perm:       0o666,
		StdoutName: "-",
		fileOpener: fileOpenerFunc(os.OpenFile),
		forceFlag:  forceFlag,
	}
	fs.BoolVar(&(o.force), forceFlag, false, fmt.Sprintf("overwrites %s if it already exists and -%s is not set.", fileHelpName, appendFlag))
	fs.BoolVar(&(o.append), appendFlag, false, fmt.Sprintf("appends to %s if it already exists.", fileHelpName))
	return o
}

func (o *Opener) openFile(filename string, extraFlags int) (io.WriteCloser, error) {
	return o.fileOpener.Open(filename, os.O_WRONLY|os.O_SYNC|os.O_CREATE|extraFlags, o.Perm)
}

func (o *Opener) openBlobStore(ctx context.Context, u *url.URL) (io.WriteCloser, error) {
	if o.append || !o.force {
		return nil, fmt.Errorf("blob store must use -%s flag", o.forceFlag)
	}

	// Separate the path from the URL as options may be present in the query
	// string.
	prefix := strings.TrimPrefix(u.Path, "/")
	u.Path = ""
	bucket := u.String()

	b, err := blob.OpenBucket(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to opening %s: %w", bucket, err)
	}
	w, err := b.NewWriter(ctx, prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("failed creating writer for %s/%s: %w", bucket, prefix, err)
	}
	return w, nil
}

// Open opens and returns a file for output with the given filename.
//
// If filename is equal to o.StdoutName, os.Stdout will be used.
// If filename does not exist, it will be created with the mode set in o.Perm.
// If filename does exist, the behavior of this function will depend on the
// flags:
//
//   - if appendFlag is set on the command line the existing file will be
//     appended to.
//   - if forceFlag is set on the command line the existing file will be
//     truncated.
//   - if neither forceFlag nor appendFlag are set an error will be
//     returned.
func (o *Opener) Open(ctx context.Context, filename string) (io.WriteCloser, error) {
	if o.StdoutName != "" && filename == o.StdoutName {
		return os.Stdout, nil
	} else if u, e := url.Parse(filename); e == nil && u.IsAbs() {
		return o.openBlobStore(ctx, u)
	}
	switch {
	case o.append:
		return o.openFile(filename, os.O_APPEND)
	case o.force:
		return o.openFile(filename, os.O_TRUNC)
	default:
		return o.openFile(filename, os.O_EXCL)
	}
}

var defaultOpener *Opener

// DefineFlags is a wrapper around CreateOpener for updating a default instance
// of Opener.
func DefineFlags(fs *flag.FlagSet, forceFlag, appendFlag, fileHelpName string) {
	defaultOpener = CreateOpener(fs, forceFlag, appendFlag, fileHelpName)
}

// Open is a wrapper around Opener.Open for the default instance of Opener.
func Open(ctx context.Context, filename string) (io.WriteCloser, error) {
	return defaultOpener.Open(ctx, filename)
}
