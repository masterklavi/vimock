package autotest

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestBlackBoxHTTPAdminAndMatchingFeatureMatrix(t *testing.T) {
	s := requireTarget(t)
	suffix := uniqueName(t)
	path := "/autotest/" + suffix + "/match"
	name := "autotest-http-" + suffix
	folder := "/autotest/http"

	mapping := map[string]any{
		"name":       name,
		"persistent": true,
		"priority":   1,
		"metadata": map[string]any{
			"wiremock-gui": map[string]any{"folder": folder},
		},
		"request": map[string]any{
			"method":  "POST",
			"urlPath": path,
			"queryParameters": map[string]any{
				"q": map[string]any{"equalTo": "1"},
			},
			"headers": map[string]any{
				"Content-Type": map[string]any{"equalTo": "application/json"},
				"X-Trace":      map[string]any{"equalTo": "abc"},
			},
			"bodyPatterns": []any{
				map[string]any{"matchesJsonPath": "$.items[?(@ == 'one')]"},
				map[string]any{"matchesJsonPath": map[string]any{"expression": "$.missing", "absent": true}},
				map[string]any{"equalToJson": map[string]any{"id": "req-1", "items": []any{"one"}}},
			},
		},
		"response": map[string]any{
			"status": 201,
			"headers": map[string]any{
				"X-Autotest": "ok",
			},
			"jsonBody": map[string]any{
				"id": "{{jsonPath request.body '$.id'}}",
				"ok": true,
			},
			"transformers": []any{"response-template"},
		},
	}
	id := createMapping(t, s, mapping)

	resp, body := s.request(t, http.MethodPost, path+"?q=1", []byte(`{"items":["one"],"id":"req-1"}`), map[string]string{
		"Content-Type": "application/json",
		"X-Trace":      "abc",
	})
	expectStatus(t, resp, body, http.StatusCreated)
	if got := resp.Header.Get("X-Autotest"); got != "ok" {
		t.Fatalf("X-Autotest = %q, want ok", got)
	}
	if !bytes.Contains(body, []byte(`"id":"req-1"`)) || !bytes.Contains(body, []byte(`"ok":true`)) {
		t.Fatalf("templated response body = %s", body)
	}

	wrongQuery, wrongQueryBody := s.request(t, http.MethodPost, path+"?q=2", []byte(`{"items":["one"],"id":"req-1"}`), map[string]string{
		"Content-Type": "application/json",
		"X-Trace":      "abc",
	})
	expectStatus(t, wrongQuery, wrongQueryBody, http.StatusNotFound)

	listResp, listBody := s.request(t, http.MethodGet, "/__admin/mappings", nil, nil)
	expectStatus(t, listResp, listBody, http.StatusOK)
	list := decodeJSONBody[mappingsListResponse](t, listBody)
	foundID := findMappingByNameAndFolder(t, list, name, folder)
	if foundID != id {
		t.Fatalf("mapping id by name+folder = %s, want %s", foundID, id)
	}

	mapping["response"] = map[string]any{
		"status": 202,
		"body":   "updated",
	}
	updateMapping(t, s, id, mapping)
	updatedResp, updatedBody := s.request(t, http.MethodPost, path+"?q=1", []byte(`{"items":["one"],"id":"req-1"}`), map[string]string{
		"Content-Type": "application/json",
		"X-Trace":      "abc",
	})
	expectStatus(t, updatedResp, updatedBody, http.StatusAccepted)
	if string(updatedBody) != "updated" {
		t.Fatalf("updated body = %q", updatedBody)
	}

	deleteMapping(t, s, id)
	afterDelete, afterDeleteBody := s.request(t, http.MethodPost, path+"?q=1", []byte(`{"items":["one"],"id":"req-1"}`), map[string]string{
		"Content-Type": "application/json",
		"X-Trace":      "abc",
	})
	expectStatus(t, afterDelete, afterDeleteBody, http.StatusNotFound)
}

func TestBlackBoxPriorityPathPatternScenariosAndDelays(t *testing.T) {
	s := requireTarget(t)
	suffix := uniqueName(t)
	base := "/autotest/" + suffix

	createMapping(t, s, map[string]any{
		"name":     "autotest-priority-fallback-" + suffix,
		"priority": 10,
		"request": map[string]any{
			"method":     "ANY",
			"urlPattern": base + "/priority/.*",
		},
		"response": map[string]any{"status": 200, "body": "fallback"},
	})
	createMapping(t, s, map[string]any{
		"name":     "autotest-priority-exact-" + suffix,
		"priority": 1,
		"request": map[string]any{
			"method":  "GET",
			"urlPath": base + "/priority/item",
		},
		"response": map[string]any{
			"status":                 200,
			"body":                   "exact",
			"fixedDelayMilliseconds": 1,
			"chunkedDribbleDelay": map[string]any{
				"numberOfChunks": 2,
				"totalDuration":  1,
			},
		},
	})
	createMapping(t, s, map[string]any{
		"name": "autotest-url-path-pattern-" + suffix,
		"request": map[string]any{
			"method":         "GET",
			"urlPathPattern": base + "/path/[0-9]+",
		},
		"response": map[string]any{"status": 200, "body": "path-pattern"},
	})

	exactResp, exactBody := s.request(t, http.MethodGet, base+"/priority/item", nil, nil)
	expectStatus(t, exactResp, exactBody, http.StatusOK)
	if string(exactBody) != "exact" {
		t.Fatalf("priority exact body = %q", exactBody)
	}
	fallbackResp, fallbackBody := s.request(t, http.MethodPatch, base+"/priority/other", nil, nil)
	expectStatus(t, fallbackResp, fallbackBody, http.StatusOK)
	if string(fallbackBody) != "fallback" {
		t.Fatalf("priority fallback body = %q", fallbackBody)
	}
	patternResp, patternBody := s.request(t, http.MethodGet, base+"/path/42", nil, nil)
	expectStatus(t, patternResp, patternBody, http.StatusOK)
	if string(patternBody) != "path-pattern" {
		t.Fatalf("path pattern body = %q", patternBody)
	}

	scenario := "autotest-scenario-" + suffix
	createMapping(t, s, map[string]any{
		"scenarioName":          scenario,
		"requiredScenarioState": "Started",
		"newScenarioState":      "Done",
		"request": map[string]any{
			"method":  "GET",
			"urlPath": base + "/scenario",
		},
		"response": map[string]any{"status": 200, "body": "first"},
	})
	createMapping(t, s, map[string]any{
		"scenarioName":          scenario,
		"requiredScenarioState": "Done",
		"request": map[string]any{
			"method":  "GET",
			"urlPath": base + "/scenario",
		},
		"response": map[string]any{"status": 200, "body": "second"},
	})
	firstResp, firstBody := s.request(t, http.MethodGet, base+"/scenario", nil, nil)
	expectStatus(t, firstResp, firstBody, http.StatusOK)
	secondResp, secondBody := s.request(t, http.MethodGet, base+"/scenario", nil, nil)
	expectStatus(t, secondResp, secondBody, http.StatusOK)
	if string(firstBody) != "first" || string(secondBody) != "second" {
		t.Fatalf("scenario bodies = %q, %q", firstBody, secondBody)
	}
	resetResp, resetBody := s.request(t, http.MethodPost, "/__admin/scenarios/reset", nil, nil)
	expectStatus(t, resetResp, resetBody, http.StatusOK)
	afterResetResp, afterResetBody := s.request(t, http.MethodGet, base+"/scenario", nil, nil)
	expectStatus(t, afterResetResp, afterResetBody, http.StatusOK)
	if string(afterResetBody) != "first" {
		t.Fatalf("scenario after reset body = %q", afterResetBody)
	}
}

func TestBlackBoxLegacyFileUploadAndBodyFile(t *testing.T) {
	s := requireTarget(t)
	suffix := uniqueName(t)
	fileName := "autotest-" + suffix + ".bin"
	payload := []byte{0, 1, 2, 255, 'v', 'i'}
	uploadLegacyFile(t, s, fileName, payload)

	path := "/autotest/" + suffix + "/file"
	createMapping(t, s, map[string]any{
		"name": "autotest-body-file-" + suffix,
		"request": map[string]any{
			"method":  "GET",
			"urlPath": path,
		},
		"response": map[string]any{
			"status":       200,
			"bodyFileName": fileName,
			"headers": map[string]any{
				"Content-Type": "application/octet-stream",
			},
		},
	})

	resp, body := s.request(t, http.MethodGet, path, nil, nil)
	expectStatus(t, resp, body, http.StatusOK)
	if !bytes.Equal(body, payload) {
		t.Fatalf("body file bytes = %v, want %v", body, payload)
	}
}

func findMappingByNameAndFolder(t *testing.T, list mappingsListResponse, name, folder string) string {
	t.Helper()
	for _, mapping := range list.Mappings {
		if mapping.Name != name {
			continue
		}
		metadata := mapping.Metadata
		gui, _ := metadata["wiremock-gui"].(map[string]any)
		if gui == nil || gui["folder"] != folder {
			continue
		}
		if strings.TrimSpace(mapping.ID) == "" {
			t.Fatalf("mapping %q has empty id", name)
		}
		return mapping.ID
	}
	t.Fatalf("mapping %q in folder %q not found", name, folder)
	return ""
}
