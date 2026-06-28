package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"vimock/internal/grpcdesc"
	"vimock/internal/mapping"
	"vimock/internal/recording"
	"vimock/internal/scenario"
)

const maxMappingBodySize = 32 << 20

type adminAPI struct {
	mappings    *mapping.Store
	scenarios   *scenario.Store
	descriptors *grpcdesc.Store
	recorder    *recording.Store
}

type listMappingsResponse struct {
	Mappings []mapping.Mapping `json:"mappings"`
	Meta     metaResponse      `json:"meta"`
}

type metaResponse struct {
	Total int `json:"total"`
}

type errorResponse struct {
	Errors []apiError `json:"errors"`
}

type apiError struct {
	Title string `json:"title"`
}

func (a adminAPI) listMappings(w http.ResponseWriter, _ *http.Request) {
	mappings := a.mappings.List()
	writeJSON(w, http.StatusOK, listMappingsResponse{
		Mappings: mappings,
		Meta: metaResponse{
			Total: len(mappings),
		},
	})
}

func (a adminAPI) getMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := mapping.ValidateID(id); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	stub, ok := a.mappings.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}

	writeJSON(w, http.StatusOK, stub)
}

func (a adminAPI) createMapping(w http.ResponseWriter, r *http.Request) {
	stub, err := readMapping(w, r, "")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	created := a.mappings.Create(stub)
	a.scenarios.MappingCreated(created)
	writeJSON(w, http.StatusCreated, created)
}

func (a adminAPI) updateMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := mapping.ValidateID(id); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	existing, ok := a.mappings.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}

	stub, err := readMapping(w, r, id)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, ok := a.mappings.Replace(id, stub)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}

	a.scenarios.MappingUpdated(existing, updated)
	writeJSON(w, http.StatusOK, updated)
}

func (a adminAPI) deleteMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := mapping.ValidateID(id); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing, ok := a.mappings.Get(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}
	if ok := a.mappings.Delete(id); !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}

	a.scenarios.MappingDeleted(existing)
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (a adminAPI) resetScenarios(w http.ResponseWriter, _ *http.Request) {
	a.scenarios.Reset()
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (a adminAPI) startRecording(w http.ResponseWriter, r *http.Request) {
	spec, err := readRecordingSpec(w, r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.recorder.Start(spec); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "Recording",
		"targetBaseUrl": spec.TargetBaseURL,
	})
}

func (a adminAPI) stopRecording(w http.ResponseWriter, _ *http.Request) {
	snapshot, err := a.recorder.Stop()
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	a.activateRecordedMappings(snapshot.Mappings)
	writeJSON(w, http.StatusOK, snapshot)
}

func (a adminAPI) snapshotRecording(w http.ResponseWriter, r *http.Request) {
	spec, err := readOptionalRecordingSpec(w, r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	snapshot, err := a.recorder.Snapshot(spec)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	a.activateRecordedMappings(snapshot.Mappings)
	writeJSON(w, http.StatusOK, snapshot)
}

func (a adminAPI) activateRecordedMappings(mappings []mapping.Mapping) {
	for _, stub := range mappings {
		created := a.mappings.Create(stub)
		a.scenarios.MappingCreated(created)
	}
}

type grpcDescriptorsResponse struct {
	Descriptors []grpcdesc.FileInfo `json:"descriptors"`
	Registry    grpcdesc.Registry   `json:"registry"`
	Meta        metaResponse        `json:"meta"`
}

func (a adminAPI) listGRPCDescriptors(w http.ResponseWriter, _ *http.Request) {
	descriptors, registry := a.descriptors.List()
	writeJSON(w, http.StatusOK, grpcDescriptorsResponse{
		Descriptors: descriptors,
		Registry:    registry,
		Meta: metaResponse{
			Total: len(descriptors),
		},
	})
}

func (a adminAPI) putGRPCDescriptor(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue("fileName")
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadedFileSize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("descriptor body exceeds %d bytes", maxUploadedFileSize))
			return
		}
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("read descriptor body: %v", err))
		return
	}

	replaced, err := a.descriptors.Put(fileName, body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	descriptors, registry := a.descriptors.List()
	status := http.StatusCreated
	if replaced {
		status = http.StatusOK
	}
	writeJSON(w, status, grpcDescriptorsResponse{
		Descriptors: descriptors,
		Registry:    registry,
		Meta: metaResponse{
			Total: len(descriptors),
		},
	})
}

func (a adminAPI) deleteGRPCDescriptor(w http.ResponseWriter, r *http.Request) {
	fileName := r.PathValue("fileName")
	if ok := a.descriptors.Delete(fileName); !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No gRPC descriptor found with name %s", fileName))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{})
}

func (a adminAPI) resetGRPC(w http.ResponseWriter, _ *http.Request) {
	if _, err := a.descriptors.Reset(); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func readMapping(w http.ResponseWriter, r *http.Request, id string) (mapping.Mapping, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMappingBodySize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return mapping.Mapping{}, fmt.Errorf("mapping body exceeds %d bytes", maxMappingBodySize)
		}
		return mapping.Mapping{}, fmt.Errorf("read mapping body: %w", err)
	}

	return mapping.ParseJSONWithID(body, id)
}

func readRecordingSpec(w http.ResponseWriter, r *http.Request) (recording.Spec, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMappingBodySize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return recording.Spec{}, fmt.Errorf("recording body exceeds %d bytes", maxMappingBodySize)
		}
		return recording.Spec{}, fmt.Errorf("read recording body: %w", err)
	}
	if len(body) == 0 {
		return recording.Spec{}, fmt.Errorf("recording body is required")
	}

	var spec recording.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		return recording.Spec{}, fmt.Errorf("recording body must be a valid JSON object: %w", err)
	}
	return spec, nil
}

func readOptionalRecordingSpec(w http.ResponseWriter, r *http.Request) (recording.Spec, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMappingBodySize)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return recording.Spec{}, fmt.Errorf("recording body exceeds %d bytes", maxMappingBodySize)
		}
		return recording.Spec{}, fmt.Errorf("read recording body: %w", err)
	}
	if len(body) == 0 {
		return recording.Spec{}, nil
	}

	var spec recording.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		return recording.Spec{}, fmt.Errorf("recording body must be a valid JSON object: %w", err)
	}
	return spec, nil
}

func writeAPIError(w http.ResponseWriter, status int, title string) {
	writeJSON(w, status, errorResponse{
		Errors: []apiError{
			{Title: title},
		},
	})
}
