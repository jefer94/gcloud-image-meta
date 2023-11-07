package function

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/disintegration/imaging"
	"github.com/vmihailenco/msgpack" // Import MessagePack package

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("ImageMeta", imageMeta)
}

// Define a map of allowed MIME types and their corresponding formats
var mimesAllowed = map[string]string{
	"image/gif":    "gif",
	"image/x-icon": "ico",
	"image/jpeg":   "jpeg",
	"image/webp":   "webp",
	"image/png":    "png",
}

const metaFileExtension = ".img.meta"

type ImageData struct {
	Filename string `msgpack:"filename"`
	Bucket   string `msgpack:"bucket"`
}

type ImageShape struct {
	Shape       string `msgpack:"shape"`
	Orientation string `msgpack:"orientation"`
	Width       int    `msgpack:"width"`
	Height      int    `msgpack:"height"`
}

func sendResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	responseBytes, err := msgpack.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/msgpack")
	w.WriteHeader(statusCode)
	w.Write(responseBytes)
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	errorData := map[string]interface{}{
		"message":     message,
		"status_code": statusCode,
	}

	sendResponse(w, errorData, statusCode)
}

func imageMeta(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	imageData := &ImageData{}
	decoder := msgpack.NewDecoder(r.Body)
	if err := decoder.Decode(imageData); err != nil {
		sendError(w, "Failed to parse request data", http.StatusBadRequest)
		return
	}

	if imageData.Filename == "" || imageData.Bucket == "" {
		sendError(w, "Incorrect filename or bucket", http.StatusBadRequest)
		return
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		sendError(w, "Failed to create client", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	bucket := client.Bucket(imageData.Bucket)
	obj := bucket.Object(imageData.Filename)

	rc, err := obj.NewReader(ctx)
	if err != nil {
		sendError(w, "Failed to read source file", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	buf := make([]byte, 512)
	_, err = io.ReadFull(rc, buf)
	if err != nil {
		sendError(w, "Failed to determine MIME type", http.StatusInternalServerError)
		return
	}

	mime := http.DetectContentType(buf)

	if _, allowed := mimesAllowed[mime]; !allowed {
		sendError(w, "File type not allowed", http.StatusBadRequest)
		return
	}

	rc, err = obj.NewReader(ctx)
	if err != nil {
		sendError(w, "Failed to read source file", http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		sendError(w, "Failed to read source file", http.StatusInternalServerError)
		return
	}

	srcImage, err := imaging.Decode(bytes.NewReader(content))
	if err != nil {
		sendError(w, "Failed to decode source image", http.StatusInternalServerError)
		return
	}

	width := srcImage.Bounds().Dx()
	height := srcImage.Bounds().Dy()

	shape, orientation := calculateImageShape(width, height)

	// Create a MessagePack-serialized map to store the shape and orientation
	meta := map[string]interface{}{
		"shape":       shape,
		"orientation": orientation,
		"width":       width,
		"height":      height,
	}

	// Serialize the metadata to MessagePack
	metaBytes, err := msgpack.Marshal(meta)
	if err != nil {
		sendError(w, "Failed to create metadata", http.StatusInternalServerError)
		return
	}

	// Save the metadata to the .img.meta file
	metaObject := bucket.Object(imageData.Filename + metaFileExtension)
	wc := metaObject.NewWriter(ctx)
	if _, err := wc.Write(metaBytes); err != nil {
		sendError(w, "Failed to save metadata to .img.meta", http.StatusInternalServerError)
		return
	}
	wc.Close()

	// Respond with the image shape and orientation
	sendResponse(w, ImageShape{Shape: shape, Orientation: orientation, Width: width, Height: height}, http.StatusOK)
}

func calculateImageShape(width, height int) (string, string) {
	if width == height {
		return "Square", "Symmetrical"
	} else if width > height {
		return "Rectangle", "Landscape"
	}
	return "Rectangle", "Portrait"
}
