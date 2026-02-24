package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxUploadSize = 5 << 20 // 5 MB

// allowedMIME maps accepted MIME types to their canonical file extension.
var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// UploadHandler holds dependencies for file upload endpoints.
type UploadHandler struct {
	uploadDir string // absolute or relative directory for stored images
}

// NewUploadHandler creates an UploadHandler that stores files in uploadDir.
func NewUploadHandler(uploadDir string) *UploadHandler {
	return &UploadHandler{uploadDir: uploadDir}
}

// UploadImage handles POST /api/v1/uploads/images
//
// Expects a multipart form with a "file" field. Validates size (<=5 MB)
// and content type (JPEG, PNG, WebP). Saves with a UUID filename.
func (h *UploadHandler) UploadImage(c *gin.Context) {
	// Limit request body.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file must not exceed 5 MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid 'file' field"})
		return
	}
	defer file.Close() //nolint:errcheck

	// Validate content type.
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedMIME[contentType]
	if !ok {
		// Fall back to sniffing the first 512 bytes.
		buf := make([]byte, 512)
		n, _ := file.Read(buf) //nolint:errcheck // best-effort sniff
		detected := http.DetectContentType(buf[:n])
		ext, ok = allowedMIME[detected]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("unsupported file type %q; allowed: JPEG, PNG, WebP", contentType),
			})
			return
		}
		// Reset reader position.
		if seeker, seekOK := file.(io.Seeker); seekOK {
			_, _ = seeker.Seek(0, io.SeekStart) //nolint:errcheck
		}
	}

	// Generate a unique filename.
	filename := uuid.New().String() + ext

	destPath := filepath.Join(h.uploadDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	defer dst.Close() //nolint:errcheck

	if _, err := io.Copy(dst, file); err != nil {
		// Clean up partial file.
		_ = os.Remove(destPath) //nolint:errcheck // best-effort cleanup
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	// Build the public URL. Use the request host to construct it.
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/api/v1/uploads/images/%s", scheme, c.Request.Host, filename)

	c.JSON(http.StatusCreated, gin.H{
		"filename": filename,
		"url":      url,
	})
}

// ServeImage handles GET /api/v1/uploads/images/:filename
//
// Serves the file from the upload directory. Returns 404 if not found.
func (h *UploadHandler) ServeImage(c *gin.Context) {
	filename := c.Param("filename")

	// Sanitize: prevent path traversal.
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	filePath := filepath.Join(h.uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.File(filePath)
}
