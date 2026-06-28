package server

import (
	"net/http"
	"os"
	"testing"
)

func TestGraphQLRuntimeSemanticMatcher(t *testing.T) {
	handler := newTestHandler()
	createMappingFromFile(t, handler, "../../testdata/graphql_mapping.json")

	resp := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/graphql",
		map[string]string{"Content-Type": "application/json"},
		`{
		  "operationName": "GetHero",
		  "variables": {
		    "episode": "JEDI"
		  },
		  "query": "query GetHero($episode: Episode) { hero(episode: $episode) { friends { name } age name } }"
		}`,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	body := decodeObjectResponse(t, resp)
	data := body["data"].(map[string]any)
	hero := data["hero"].(map[string]any)
	if hero["name"] != "Luke Skywalker" {
		t.Fatalf("hero.name = %v, want Luke Skywalker", hero["name"])
	}
}

func TestGraphQLRuntimeRejectsVariablesArrayOrderAndUnexpectedOperationName(t *testing.T) {
	handler := newTestHandler()
	createMapping(t, handler, `{
	  "request": {
	    "method": "POST",
	    "urlPathPattern": "/graphql",
	    "customMatcher": {
	      "name": "graphql-body-matcher",
	      "parameters": {
	        "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { name age } }",
	        "variables": {
	          "ids": [1, 2, 3]
	        }
	      }
	    }
	  },
	  "response": {
	    "status": 200,
	    "body": "matched"
	  }
	}`)

	differentArrayOrder := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/graphql",
		map[string]string{"Content-Type": "application/json"},
		`{
		  "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { age name } }",
		  "variables": {
		    "ids": [3, 2, 1]
		  }
		}`,
	)
	if differentArrayOrder.Code != http.StatusNotFound {
		t.Fatalf("array order status = %d, want 404", differentArrayOrder.Code)
	}

	unexpectedOperationName := requestWithHeadersAndBody(
		t,
		handler,
		http.MethodPost,
		"/graphql",
		map[string]string{"Content-Type": "application/json"},
		`{
		  "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { age name } }",
		  "variables": {
		    "ids": [1, 2, 3]
		  },
		  "operationName": "GetCharacters"
		}`,
	)
	if unexpectedOperationName.Code != http.StatusNotFound {
		t.Fatalf("unexpected operationName status = %d, want 404", unexpectedOperationName.Code)
	}
}

func createMappingFromFile(t *testing.T, handler http.Handler, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mapping %s: %v", path, err)
	}
	return createMapping(t, handler, string(body))
}
