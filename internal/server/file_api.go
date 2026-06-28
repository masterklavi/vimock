package server

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vimock/internal/files"
	"vimock/internal/grpcdesc"
)

const (
	legacyFileAuthToken = "vimock-file-token"
	maxUploadedFileSize = 64 << 20
)

type fileAPI struct {
	files       files.Store
	descriptors *grpcdesc.Store
}

func (a fileAPI) login(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(legacyFileAuthToken))
}

func (a fileAPI) createUpload(w http.ResponseWriter, r *http.Request) {
	if !authorizeLegacyFileRequest(w, r) {
		return
	}
	if err := validateTusCreateRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := uploadFileName(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.files.Put(fileName, nil)
	w.Header().Set("Tus-Resumable", "1.0.0")
	w.Header().Set("Upload-Offset", "0")
	w.Header().Set("Location", "/api/tus/"+url.PathEscape(fileName))
	w.WriteHeader(http.StatusCreated)
}

func (a fileAPI) patchUpload(w http.ResponseWriter, r *http.Request) {
	if !authorizeLegacyFileRequest(w, r) {
		return
	}
	if err := validateTusPatchRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileName, err := validateUploadFileName(r.PathValue("file"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadedFileSize)
	defer r.Body.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, fmt.Sprintf("uploaded file exceeds %d bytes", maxUploadedFileSize), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, fmt.Sprintf("read uploaded file: %v", err), http.StatusBadRequest)
		return
	}

	a.files.Put(fileName, data)
	if a.descriptors != nil {
		a.descriptors.PutLegacy(fileName, data)
	}
	w.Header().Set("Tus-Resumable", "1.0.0")
	w.Header().Set("Upload-Offset", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusNoContent)
}

func authorizeLegacyFileRequest(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Auth") == legacyFileAuthToken {
		return true
	}

	http.Error(w, "invalid X-Auth token", http.StatusUnauthorized)
	return false
}

func validateTusCreateRequest(r *http.Request) error {
	if r.URL.Query().Get("override") != "true" {
		return fmt.Errorf("override=true is required")
	}
	if r.Header.Get("Tus-Resumable") != "1.0.0" {
		return fmt.Errorf("Tus-Resumable=1.0.0 is required")
	}

	uploadLength := strings.TrimSpace(r.Header.Get("Upload-Length"))
	if uploadLength == "" {
		return fmt.Errorf("Upload-Length is required")
	}
	if _, err := strconv.ParseInt(uploadLength, 10, 64); err != nil {
		return fmt.Errorf("Upload-Length must be an integer")
	}

	return nil
}

func validateTusPatchRequest(r *http.Request) error {
	if r.URL.Query().Get("override") != "true" {
		return fmt.Errorf("override=true is required")
	}
	if r.Header.Get("Tus-Resumable") != "1.0.0" {
		return fmt.Errorf("Tus-Resumable=1.0.0 is required")
	}
	if r.Header.Get("Upload-Offset") != "0" {
		return fmt.Errorf("only Upload-Offset=0 is supported")
	}

	return nil
}

func uploadFileName(r *http.Request) (string, error) {
	if name, ok, err := uploadMetadataFileName(r.Header.Get("Upload-Metadata")); ok || err != nil {
		if err != nil {
			return "", err
		}
		return validateUploadFileName(name)
	}

	return validateUploadFileName(r.PathValue("file"))
}

func uploadMetadataFileName(metadata string) (string, bool, error) {
	for _, entry := range strings.Split(metadata, ",") {
		fields := strings.Fields(strings.TrimSpace(entry))
		if len(fields) < 2 || fields[0] != "filename" {
			continue
		}

		decoded, err := hex.DecodeString(fields[1])
		if err != nil {
			return "", true, fmt.Errorf("decode Upload-Metadata filename: %w", err)
		}
		return string(decoded), true, nil
	}

	return "", false, nil
}

func validateUploadFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("file name is required")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("invalid file name %q", name)
	}

	return name, nil
}
