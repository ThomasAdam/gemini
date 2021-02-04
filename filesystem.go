package gemini

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// A Dir implements FileSystem using the native file system restricted to a
// specific directory tree.
//
// While the FileSystem.Open method takes '/'-separated paths, a Dir's string
// value is a filename on the native file system, not a URL, so it is separated
// by filepath.Separator, which isn't necessarily '/'.
//
// Note that Dir could expose sensitive files and directories. Dir will follow
// symlinks pointing out of the directory tree, which can be especially
// dangerous if serving from a directory in which users are able to create
// arbitrary symlinks. Dir will also allow access to files and directories
// starting with a period, which could expose sensitive directories like .git or
// sensitive files like .htpasswd. To exclude files with a leading period,
// remove the files/directories from the server or create a custom FileSystem
// implementation.
//
// An empty Dir is treated as ".".
type Dir string

// Open implements FileSystem using os.Open, opening files for reading rooted
// and relative to the directory d.
func (d Dir) Open(name string) (File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}

	dir := string(d)
	if dir == "" {
		dir = "."
	}

	fullName := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, err := os.Open(fullName)
	if err != nil {
		return nil, err
	}

	return f, nil

}

// A File is returned by a FileSystem's Open method and can be served by the FileServer implementation.
//
// The methods should behave the same as those on an *os.File.
type File interface {
	io.Closer
	io.Reader
	io.Seeker
	Readdir(count int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

// A FileSystem implements access to a collection of named files. The elements
// in a file path are separated by slash ('/', U+002F) characters, regardless of
// host operating system convention.
type FileSystem interface {
	Open(name string) (File, error)
}

type fileHandler struct {
	root FileSystem
}

// FileServer returns a handler that serves HTTP requests with the contents of
// the file system rooted at root.
//
// To use the operating system's file system implementation, use gemini.Dir:
//
//     gemini.Handle("/", gemini.FileServer(gemini.Dir("/tmp")))
//
// Once go 1.16 is released, this will most likely be dropped in favor of the
// built-in FS interfaces.
func FileServer(root FileSystem) Handler {
	return &fileHandler{root}
}

func (f *fileHandler) ServeGemini(ctx context.Context, r *Request) *Response {
	upath := r.URL.Path

	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	return serveFile(ctx, r, f.root, cleanPath(upath))
}

// name is '/'-separated, not filepath.Separator.
func serveFile(ctx context.Context, r *Request, fs FileSystem, name string) (resp *Response) {
	const indexPage = "/index.gmi"

	f, err := fs.Open(name)
	if err != nil {
		resp = NewResponse(StatusPermanentFailure, err.Error())
		return
	}
	defer func() {
		if resp == nil || resp.Body == nil {
			f.Close()
		}
	}()

	d, err := f.Stat()
	if err != nil {
		resp = NewResponse(StatusPermanentFailure, err.Error())
		return
	}

	// redirect to canonical path: / at end of directory url
	// r.URL.Path always begins with /
	pathName := r.URL.Path
	if d.IsDir() {
		if pathName[len(pathName)-1] != '/' {
			return NewResponse(StatusRedirect, path.Base(pathName)+"/")
		}
	} else {
		if pathName[len(pathName)-1] == '/' {
			return NewResponse(StatusRedirect, "../"+path.Base(pathName))
		}
	}

	if d.IsDir() {
		// use contents of index.gmi for directory, if present
		index := strings.TrimSuffix(name, "/") + indexPage
		ff, err := fs.Open(index)
		if err == nil {
			dd, err := ff.Stat()
			if err == nil {
				// Close the old file because we're going to overwrite the
				// reference.
				_ = f.Close()

				name = index
				d = dd
				f = ff
			}
		}
	}

	// Still a directory? (we didn't find an index.gmi file)
	if d.IsDir() {
		entries, err := f.Readdir(0)
		if err != nil {
			resp = NewResponse(StatusPermanentFailure, err.Error())
			return
		}

		// Sort all items, directories first
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() == entries[j].IsDir() {
				return entries[i].Name() < entries[j].Name()
			}

			return entries[i].IsDir()
		})

		buf := bytes.NewBuffer(nil)

		for _, entry := range entries {
			buf.WriteString("=> ")
			buf.WriteString(url.PathEscape(entry.Name()))
			if entry.IsDir() {
				buf.WriteString("/")
			}
			buf.WriteString("\n")
		}

		return NewResponseBody(StatusSuccess, "text/gemini", ioutil.NopCloser(buf))
	}

	mimeType := mime.TypeByExtension(path.Ext(d.Name()))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return NewResponseBody(StatusSuccess, mimeType, f)
}
