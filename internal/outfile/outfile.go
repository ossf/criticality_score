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

// fileOpenFunc makes it possible to mock os.OpenFile() for testing.
type fileOpenFunc func(string, int, os.FileMode) (*os.File, error)

// A FilenameTransform applies a transform to the filename before it used to
// open a file.
type FilenameTransform func(string) string

// NameWriteCloser implements the io.WriteCloser interface, but also allows a
// a name to be returned so that the object can be identified.
type NameWriteCloser interface {
	// Name returns a name for the Writer.
	Name() string
	io.WriteCloser
}

type writeCloserWrapper struct {
	io.WriteCloser
	name string
}

// Name implements the NameWriteCloser interface.
func (w *writeCloserWrapper) Name() string {
	if w == nil {
		return ""
	}
	return w.name
}

type Opener struct {
	FilenameTransform FilenameTransform
	fileOpener        fileOpenFunc
	filename          string
	forceFlag         string
	force             bool
	append            bool
	Perm              os.FileMode
}

// CreateOpener creates an Opener and defines the sepecified flags fileFlag, forceFlag and appendFlag.
func CreateOpener(fs *flag.FlagSet, fileFlag, forceFlag, appendFlag, fileHelpName string) *Opener {
	o := &Opener{
		Perm:              0o666,
		FilenameTransform: func(f string) string { return f },
		fileOpener:        os.OpenFile,
		forceFlag:         forceFlag,
	}
	fs.StringVar(&(o.filename), fileFlag, "", fmt.Sprintf("use the file `%s` for output. Defaults to stdout if not set.", fileHelpName))
	fs.BoolVar(&(o.force), forceFlag, false, fmt.Sprintf("overwrites %s if it already exists and -%s is not set.", fileHelpName, appendFlag))
	fs.BoolVar(&(o.append), appendFlag, false, fmt.Sprintf("appends to %s if it already exists.", fileHelpName))
	return o
}

func (o *Opener) openFile(filename string, extraFlags int) (NameWriteCloser, error) {
	return o.fileOpener(filename, os.O_WRONLY|os.O_SYNC|os.O_CREATE|extraFlags, o.Perm)
}

func (o *Opener) openBlobStore(ctx context.Context, u *url.URL) (NameWriteCloser, error) {
	if o.append || !o.force {
		return nil, fmt.Errorf("blob store must use -%s flag", o.forceFlag)
	}
	name := u.String()

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
	return &writeCloserWrapper{
		WriteCloser: w,
		name:        name,
	}, nil
}

// Open opens and returns a file for output with the filename set by the fileFlag,
// applying any transform set in FilenameTransform.
//
// If filename is empty/unset, os.Stdout will be used.
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
func (o *Opener) Open(ctx context.Context) (NameWriteCloser, error) {
	f := o.FilenameTransform(o.filename)
	if f == "" {
		return os.Stdout, nil
	} else if u, e := url.Parse(f); e == nil && u.IsAbs() {
		return o.openBlobStore(ctx, u)
	}
	switch {
	case o.append:
		return o.openFile(f, os.O_APPEND)
	case o.force:
		return o.openFile(f, os.O_TRUNC)
	default:
		return o.openFile(f, os.O_EXCL)
	}
}

var DefaultOpener *Opener

// DefineFlags is a wrapper around CreateOpener for updating a default instance
// of Opener.
func DefineFlags(fs *flag.FlagSet, fileFlag, forceFlag, appendFlag, fileHelpName string) {
	DefaultOpener = CreateOpener(fs, fileFlag, forceFlag, appendFlag, fileHelpName)
}

// Open is a wrapper around Opener.Open for the default instance of Opener.
//
// Must only be called after DefineFlags.
func Open(ctx context.Context) (NameWriteCloser, error) {
	return DefaultOpener.Open(ctx)
}
