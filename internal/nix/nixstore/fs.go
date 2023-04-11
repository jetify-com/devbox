package nixstore

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// S3BucketFS is an [fs.FS] for an AWS S3 bucket.
type S3BucketFS struct {
	BucketURL *url.URL
	Client    http.Client
}

// NewS3BucketFS returns a file system for the S3 bucket at the URL bucketURL.
func NewS3BucketFS(bucketURL string) (*S3BucketFS, error) {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &S3BucketFS{BucketURL: u}, nil
}

// Open makes an HTTP request to read a file from the bucket. Once a file is
// open it is subject to any connection read timeouts, including those set by
// S3.
func (s3 *S3BucketFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	fi, body, err := s3.doRequest(http.MethodGet, name)
	if err != nil {
		return nil, err
	}
	return &S3File{
		respBody: body,
		info:     fi,
	}, nil
}

func (s3 *S3BucketFS) doRequest(method, filename string) (info *S3FileInfo, body io.ReadCloser, err error) {
	u := s3.BucketURL.JoinPath(filename).String()
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	resp, err := s3.Client.Do(req)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	defer func() {
		// Consume the body if we're not returning it so that the
		// http.Client can reuse the underlying connection.
		if body == nil {
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
		}

		// Wrap all errors in a *fs.PathError per the fs.FS docs.
		if err != nil {
			err = &fs.PathError{
				Op:   "open",
				Path: filename,
				Err:  err,
			}
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		if req.Method == http.MethodHead {
			return parseS3FileInfo(resp), nil, nil
		}
		return parseS3FileInfo(resp), resp.Body, nil
	case http.StatusNotFound:
		return nil, nil, fs.ErrNotExist
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, nil, fs.ErrPermission
	default:
		if s3Err := tryUnmarshalS3Error(resp.Body); s3Err != nil {
			return nil, nil, errors.Errorf("%s %q: response status %q: %v",
				req.Method, req.URL, resp.Status, s3Err)
		}
		return nil, nil, errors.Errorf("%s %q: response status %q",
			req.Method, req.URL, resp.Status)
	}
}

// parseS3FileInfo creates an S3FileInfo from an [http.Response]. If the
// response is missing some headers, then S3FileInfo will contain what
// information is available.
func parseS3FileInfo(resp *http.Response) *S3FileInfo {
	return &S3FileInfo{
		URL:           resp.Request.URL,
		LastModified:  resp.Header.Get("Last-Modified"),
		Etag:          resp.Header.Get("Etag"),
		ContentLength: resp.ContentLength,
		ContentType:   resp.Header.Get("Content-Type"),
	}
}

// S3File is an [fs.File] opened by an [S3BucketFS].
type S3File struct {
	respBody io.ReadCloser
	info     *S3FileInfo
}

// Stat returns the underlying S3 object metadata as an [fs.FileInfo]. The info
// will always be of type [S3FileInfo].
//
// A call to Stat does not issue a network request.
func (f *S3File) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

// Read reads the file's contents from S3.
func (f *S3File) Read(p []byte) (n int, err error) {
	n, err = f.respBody.Read(p)
	if err == io.EOF { //nolint:errorlint
		// Don't wrap io.EOF per io.Writer.
		return n, err
	}
	return n, err
}

// Close closes the file and frees any underlying network connections.
func (f *S3File) Close() error {
	io.Copy(io.Discard, f.respBody) //nolint:errcheck
	return f.respBody.Close()
}

// S3FileInfo is an [fs.FileInfo] containing the S3 object metadata for an
// [S3File].
type S3FileInfo struct {
	// URL is the full URL for downloading the underlying S3 object.
	URL *url.URL

	// LastModified is the raw LastModified attribute of the S3 object.
	LastModified string

	// Etag is the HTTP Etag header from the S3 API response. It may be
	// empty if the response didn't include one.
	Etag string

	// ContentLength is the [http.Response.ContentLength] from the S3 API
	// response. It can be -1 if the response Content-Length header is
	// missing.
	ContentLength int64

	// ContentType is the HTTP Content-Type header from the S3 API response.
	ContentType string
}

// Name interprets the S3 file's key as a path and returns its base name.
func (fi *S3FileInfo) Name() string { return path.Base(fi.URL.Path) }

// Mode is currently always 0444.
func (fi *S3FileInfo) Mode() fs.FileMode { return 0444 }

// IsDir currently always returns false.
func (fi *S3FileInfo) IsDir() bool { return false }

// Sys always returns nil.
func (fi *S3FileInfo) Sys() any { return nil }

// Size returns the S3 file's size, which might be 0 if it is unknown.
func (fi *S3FileInfo) Size() int64 {
	if fi.ContentLength < 0 {
		return 0
	}
	return fi.ContentLength
}

// ModTime parses and returns the S3 file's last modified time. It returns the
// zero time if there's a parse error or the time is unknown.
func (fi *S3FileInfo) ModTime() time.Time {
	if fi.LastModified == "" {
		return time.Time{}
	}
	modTime, err := http.ParseTime(fi.LastModified)
	if err != nil {
		return time.Time{}
	}
	return modTime
}

// tryUnmarshalS3Error unmarshals the body of an S3 error response into a Go
// error. If there's an error reading or unmarshaling the body,
// tryUnmarshalS3Error returns nil.
//
//nolint:nilerr
func tryUnmarshalS3Error(r io.Reader) error {
	s3Err := struct {
		Code     string
		Message  string
		Resource string
	}{}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	if err := xml.Unmarshal(b, &s3Err); err != nil {
		return nil
	}
	return errors.Errorf("got S3 error code %q when requesting %q: %s",
		s3Err.Code, s3Err.Resource, s3Err.Message)
}

// readLinkFS is an [os.DirFS] that supports reading symlinks. It satisifies
// the interface discussed in the [accepted Go proposal for fs.ReadLinkFS].
// If differs from the proposed implementation in that it allows absolute
// symlinks by translating them to relative paths.
//
// [accepted Go proposal for fs.ReadLinkFS]: https://github.com/golang/go/issues/49580
type readLinkFS struct {
	fs.FS
	dir string
}

func newReadLinkFS(dir string) fs.FS {
	return &readLinkFS{FS: os.DirFS(dir), dir: dir}
}

func (fsys *readLinkFS) ReadLink(name string) (string, error) {
	osName := filepath.Join(fsys.dir, filepath.FromSlash(name))
	dst, err := os.Readlink(osName)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(filepath.Dir(osName), dst)
	}
	if filepath.IsAbs(dst) {
		dst, err = filepath.Rel(fsys.dir, dst)
		if err != nil {
			return "", fmt.Errorf("%s evaluates to a path outside of the root", name)
		}
	}
	if !filepath.IsLocal(dst) {
		return "", fmt.Errorf("%s evaluates to a path outside of the root", name)
	}
	return dst, nil
}

// readLink returns the destination of a symbolic link. If the file system
// doesn't implement ReadLink, then it returns an error. It matches the
// interface discussed in the [accepted Go proposal for fs.ReadLink].
//
// [accepted Go proposal for fs.ReadLink]: https://github.com/golang/go/issues/49580
func readLink(fsys fs.FS, name string) (string, error) {
	rlFS, ok := fsys.(interface{ ReadLink(string) (string, error) })
	if !ok {
		return "", &fs.PathError{
			Op:   "readlink",
			Path: name,
			Err:  errors.New("not implemented"),
		}
	}
	return rlFS.ReadLink(name)
}

// readDirUnsorted acts identically to [fs.ReadDir] except that it skips
// sorting the directory entries when possible to save some time.
func readDirUnsorted(fsys fs.FS, path string) ([]fs.DirEntry, error) {
	if fsys, ok := fsys.(fs.ReadDirFS); ok {
		return fsys.ReadDir(path)
	}
	f, err := fsys.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dir, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{
			Op:   "readdir",
			Path: path,
			Err:  errors.New("not implemented"),
		}
	}
	return dir.ReadDir(-1)
}
