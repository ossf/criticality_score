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
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
)

type openCall struct {
	filename string
	flags    int
	perm     os.FileMode
}

type testOpener struct {
	flag     *flag.FlagSet
	openErr  error
	lastOpen *openCall
	opener   *Opener
}

func newTestOpener(t *testing.T) *testOpener {
	t.Helper()
	o := &testOpener{}
	o.flag = flag.NewFlagSet("", flag.ContinueOnError)
	o.opener = CreateOpener(o.flag, "out", "force", "append", "FILE")
	o.opener.Perm = 0o567
	o.opener.fileOpener = func(filename string, flags int, perm os.FileMode) (*os.File, error) {
		o.lastOpen = &openCall{
			filename: filename,
			flags:    flags,
			perm:     perm,
		}
		if o.openErr != nil {
			return nil, o.openErr
		} else {
			dir := t.TempDir()
			cwd, err := os.Getwd() // Save the CWD so we can restore it later.
			if err != nil {
				return nil, err
			}
			if err := os.Chdir(dir); err != nil {
				return nil, err
			}
			defer os.Chdir(cwd) // Restore the CWD so the temp dir can be cleaned up on Windows.
			return os.Create(filename)
		}
	}
	return o
}

func TestForceFlagDefined(t *testing.T) {
	o := newTestOpener(t)
	f := o.flag.Lookup("force")
	if f == nil {
		t.Fatal("Lookup() == nil, wanted a flag.")
	}
}

func TestAppendFlagDefined(t *testing.T) {
	o := newTestOpener(t)
	f := o.flag.Lookup("append")
	if f == nil {
		t.Fatal("Lookup() == nil, wanted a flag.")
	}
}

func TestOpenStdout(t *testing.T) {
	o := newTestOpener(t)
	f, err := o.opener.Open(context.Background())
	if err != nil {
		t.Fatalf("Open() == %v, want nil", err)
	}
	if f != os.Stdout {
		t.Fatal("Open() == not stdout, want stdout")
	}
}

func TestOpenBucketUrl(t *testing.T) {
	o := newTestOpener(t)
	o.flag.Parse([]string{"-force", "-out=mem://bucket/prefix"})
	f, err := o.opener.Open(context.Background())
	if err != nil {
		t.Fatalf("Open() == %v, want nil", err)
	}
	defer f.Close()
	if o.lastOpen != nil {
		t.Fatal("Open(...) called instead of bucket code")
	}
	if f == nil {
		t.Fatal("Open() == nil, want io.WriterCloser")
	}
}

func TestOpenBucketUrlNoForceFlag(t *testing.T) {
	o := newTestOpener(t)
	o.flag.Parse([]string{"-out=mem://bucket/prefix"})
	f, err := o.opener.Open(context.Background())
	if err == nil {
		defer f.Close()
		t.Fatalf("Open() == nil, want an error")
	}
}

func TestOpenFlagTest(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name         string
		args         []string
		openErr      error
		expectedFlag int
	}{
		{
			name:         "no args",
			args:         []string{},
			expectedFlag: os.O_EXCL,
		},
		{
			name:         "append only flag",
			args:         []string{"-append"},
			expectedFlag: os.O_APPEND,
		},
		{
			name:         "force only flag",
			args:         []string{"-force"},
			expectedFlag: os.O_TRUNC,
		},
		{
			name:         "both flags",
			args:         []string{"-force", "-append"},
			expectedFlag: os.O_APPEND,
		},
	}

	// Test success responses
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := newTestOpener(t)
			o.flag.Parse(append(test.args, "-out=testfile"))
			f, err := o.opener.Open(context.Background())
			if err != nil {
				t.Fatalf("Open() == %v, want nil", err)
			}
			defer f.Close()
			if f == nil {
				t.Fatal("Open() == nil, want a file")
			}
			if got := f.Name(); got != "testfile" {
				t.Fatalf("Open().Name() == %s; want %s", got, "testfile")
			}
			assertLastOpen(t, o, "testfile", test.expectedFlag, 0o567)
		})
	}

	// Test error responses
	for _, test := range tests {
		t.Run(test.name+" error", func(t *testing.T) {
			o := newTestOpener(t)
			o.flag.Parse(append(test.args, "-out=testfile"))
			o.openErr = errors.New("test error")
			f, err := o.opener.Open(context.Background())
			if err == nil {
				defer f.Close()
				t.Fatalf("Open() is nil, want %v", o.openErr)
			}
		})
	}
}

func assertLastOpen(t *testing.T, o *testOpener, filename string, requireFlags int, perm os.FileMode) {
	t.Helper()
	if o.lastOpen == nil {
		t.Fatalf("Open(...) not called, want call to Open(...)")
	}
	if o.lastOpen.filename != filename {
		t.Fatalf("Open(%v, _, _) called, want Open(%v, _, _)", o.lastOpen.filename, filename)
	}
	if o.lastOpen.flags&requireFlags != requireFlags {
		t.Fatalf("Open(_, %v, _) called, want Open(_, %v, _)", o.lastOpen.flags&requireFlags, requireFlags)
	}
	if o.lastOpen.perm != perm {
		t.Fatalf("Open(_, _, %v) called, want Open(_, _, %v)", o.lastOpen.perm, perm)
	}
}

func TestFilenameTransform(t *testing.T) {
	want := "prefix-testfile-suffix"
	o := newTestOpener(t)
	o.opener.FilenameTransform = func(f string) string { return fmt.Sprintf("prefix-%s-suffix", f) }
	o.flag.Parse([]string{"-out=testfile"})
	f, err := o.opener.Open(context.Background())
	if err != nil {
		t.Fatalf("Open() == %v, want nil", err)
	}
	defer f.Close()
	if f == nil {
		t.Fatal("Open() == nil, want a file")
	}
	if got := f.Name(); got != want {
		t.Fatalf("Open().Name() == %s; want %s", got, want)
	}
	assertLastOpen(t, o, want, os.O_EXCL, 0o567)
}
