package api

import (
	"crypto/rand"
	"encoding/hex"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	// Register the image decoders so image.DecodeConfig can read the
	// dimensions of every format the uploader accepts. Blank imports: they are
	// used for their registration side effect, not their exported names.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/biletflow/api/internal/httpx"
)

// Upload limits. An event banner is a photograph, not a video: 5 MB is
// generous for one and refuses the other outright.
const (
	maxUploadBytes = 5 << 20
	uploadFormKey  = "file"

	// Image dimension bounds. Small enough at the bottom that a favicon or a
	// stray icon is refused as a banner, large enough at the top for a real
	// poster, and the megapixel cap refuses a "decompression bomb" - a small
	// file that expands to gigabytes of pixels in memory - even when its width
	// and height are individually within range.
	minImageDimension = 200
	maxImageDimension = 6000
	maxImagePixels    = 40_000_000
)

// CodeBadImageDimensions lets the UI explain a dimension failure precisely.
const CodeBadImageDimensions = "invalid_image_dimensions"

// CodeUploadTooLarge and CodeUnsupportedMedia let the UI say which rule was
// broken rather than "upload failed".
const (
	CodeUploadTooLarge   = "file_too_large"
	CodeUnsupportedMedia = "unsupported_media_type"
)

// allowedUploadTypes maps a sniffed content type to the extension it is stored
// under.
//
// Images for event banners (SRS 4.2), plus PDF because a support attachment is
// as often a receipt or a printed ticket as a screenshot (SRS 4.13).
//
// The type is detected from the file's own first bytes, not from the
// Content-Type header or the filename, because both are supplied by the client
// and neither is evidence of anything.
var allowedUploadTypes = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
}

type uploadResponse struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Bytes    int64  `json:"bytes"`
	MimeType string `json:"mime_type"`
}

// handleUploadImage stores an event banner and returns the URL to reference it
// by (SRS 4.2, "add ... images").
//
// Files land on local disk. A real deployment would put them in S3 or an
// equivalent; the seam is this handler and the static route that serves the
// directory, so swapping the destination changes one file.
func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	// MaxBytesReader makes an oversized upload fail while reading rather than
	// after the whole thing has been buffered into memory.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1024)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, CodeUploadTooLarge,
			"That image is too large. The limit is 5 MB.")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile(uploadFormKey)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeValidationFailed,
			`Attach the image as a multipart field named "file".`)
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxUploadBytes {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, CodeUploadTooLarge,
			"That image is too large. The limit is 5 MB.")
		return
	}

	// Sniff the type from the content itself.
	sniff := make([]byte, 512)
	n, err := io.ReadFull(file, sniff)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		httpx.WriteInternalError(w, r, err)
		return
	}
	sniff = sniff[:n]

	mimeType := http.DetectContentType(sniff)
	extension, ok := allowedUploadTypes[mimeType]
	if !ok {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, CodeUnsupportedMedia,
			"Upload a JPEG, PNG, GIF or WebP image, or a PDF.")
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// A PDF has no pixel dimensions; every image type does, and an image that is
	// too small, too large, or a decompression bomb is refused here rather than
	// stored and served as a broken banner (SRS 4.2).
	if mimeType != "application/pdf" {
		if msg := checkImageDimensions(file); msg != "" {
			httpx.WriteError(w, http.StatusUnprocessableEntity, CodeBadImageDimensions, msg)
			return
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			httpx.WriteInternalError(w, r, err)
			return
		}
	}

	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	// The stored name is random, never the uploaded one. A client-supplied
	// filename is a path-traversal vector and a way to overwrite somebody
	// else's banner; neither is worth the cosmetic benefit of keeping it.
	name, err := randomFilename(extension)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	destination := filepath.Join(s.cfg.UploadDir, name)
	out, err := os.Create(destination)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	written, err := io.Copy(out, io.LimitReader(file, maxUploadBytes))
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(destination)
		httpx.WriteInternalError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, uploadResponse{
		URL:      s.cfg.APIBaseURL + uploadURLPrefix + name,
		Filename: name,
		Bytes:    written,
		MimeType: mimeType,
	})
}

// checkImageDimensions reads only the image header (not the whole file) and
// returns a human message when the dimensions are out of bounds, or "" when
// they are acceptable. It leaves the reader wherever DecodeConfig stopped; the
// caller re-seeks before storing.
func checkImageDimensions(r io.Reader) string {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return "That image could not be read. Upload a valid JPEG, PNG, GIF or WebP."
	}
	if cfg.Width < minImageDimension || cfg.Height < minImageDimension {
		return "That image is too small. It must be at least 200×200 pixels."
	}
	if cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
		return "That image is too large. Each side must be at most 6000 pixels."
	}
	if cfg.Width*cfg.Height > maxImagePixels {
		return "That image has too many pixels. Keep it under 40 megapixels."
	}
	return ""
}

// uploadURLPrefix is where stored files are served from.
const uploadURLPrefix = "/uploads/"

func randomFilename(extension string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw) + extension, nil
}

// uploadsHandler serves the upload directory.
//
// It is wrapped rather than handed straight to http.FileServer so that a
// request for a directory - or for anything that is not a plain file - is a
// 404 instead of an index of everybody's banners.
func (s *Server) uploadsHandler() http.Handler {
	fileServer := http.FileServer(http.Dir(s.cfg.UploadDir))

	return http.StripPrefix(uploadURLPrefix, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			name := strings.TrimPrefix(r.URL.Path, "/")

			// Only the flat names this API generates are servable. Anything
			// with a separator in it is either a traversal attempt or a
			// mistake, and neither should reach the filesystem.
			if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No such file.")
				return
			}

			info, err := os.Stat(filepath.Join(s.cfg.UploadDir, name))
			if err != nil || info.IsDir() {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "No such file.")
				return
			}

			// Banners are immutable: the name is a random string that is never
			// reused, so a long cache is safe and saves re-fetching a poster on
			// every page view.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
		}))
}
