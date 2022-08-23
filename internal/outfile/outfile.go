package outfile

import (
	"flag"
	"fmt"
	"os"
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
	}
	fs.BoolVar(&(o.force), forceFlag, false, fmt.Sprintf("overwrites %s if it already exists and -%s is not set.", fileHelpName, appendFlag))
	fs.BoolVar(&(o.append), appendFlag, false, fmt.Sprintf("appends to %s if it already exists.", fileHelpName))
	return o
}

func (o *Opener) openInternal(filename string, extraFlags int) (*os.File, error) {
	return o.fileOpener.Open(filename, os.O_WRONLY|os.O_SYNC|os.O_CREATE|extraFlags, o.Perm)
}

// Open opens and returns a file for output with the given filename.
//
// If filename is equal to o.StdoutName, os.Stdout will be used.
// If filename does not exist, it will be created with the mode set in o.Perm.
// If filename does exist, the behavior of this function will depend on the
// flags:
// - if appendFlag is set on the command line the existing file will be
//
//	appended to.
//
// - if forceFlag is set on the command line the existing file will be
//
//	truncated.
//
// - if neither forceFlag nor appendFlag are set an error will be
//
//	returned.
func (o *Opener) Open(filename string) (f *os.File, err error) {
	if o.StdoutName != "" && filename == o.StdoutName {
		f = os.Stdout
	} else if o.append {
		f, err = o.openInternal(filename, os.O_APPEND)
	} else if o.force {
		f, err = o.openInternal(filename, os.O_TRUNC)
	} else {
		f, err = o.openInternal(filename, os.O_EXCL)
	}
	return
}

var defaultOpener *Opener

// DefineFlags is a wrapper around CreateOpener for updating a default instance
// of Opener.
func DefineFlags(fs *flag.FlagSet, forceFlag, appendFlag, fileHelpName string) {
	defaultOpener = CreateOpener(fs, forceFlag, appendFlag, fileHelpName)
}

// Open is a wrapper around Opener.Open for the default instance of Opener.
func Open(filename string) (*os.File, error) {
	return defaultOpener.Open(filename)
}
