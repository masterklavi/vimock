package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"vimock/internal/mapping"
)

const maxMappingBodySize = 32 << 20

type adminAPI struct {
	mappings *mapping.Store
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
	writeJSON(w, http.StatusCreated, created)
}

func (a adminAPI) updateMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := mapping.ValidateID(id); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := a.mappings.Get(id); !ok {
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

	writeJSON(w, http.StatusOK, updated)
}

func (a adminAPI) deleteMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := mapping.ValidateID(id); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if ok := a.mappings.Delete(id); !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("No stub mapping found with id %s", id))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{})
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

func writeAPIError(w http.ResponseWriter, status int, title string) {
	writeJSON(w, status, errorResponse{
		Errors: []apiError{
			{Title: title},
		},
	})
}
