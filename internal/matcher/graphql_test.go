package matcher

import (
	"encoding/json"
	"testing"
)

func TestGraphQLBodyMatcherMatchesNormalizedQueryVariablesAndOperationName(t *testing.T) {
	matcher := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "query GetHero($episode: Episode) { hero(episode: $episode) { age name friends { id name } } }",
	    "variables": {
	      "episode": "JEDI"
	    },
	    "operationName": "GetHero"
	  }
	}`)

	body := []byte(`{
	  "operationName": "GetHero",
	  "variables": {
	    "episode": "JEDI"
	  },
	  "query": "query GetHero($episode: Episode){ hero(episode:$episode){ friends { name id } name age } }"
	}`)
	if !matcher.Matches(NewBodyContext(body)) {
		t.Fatalf("GraphQL matcher did not match semantically equivalent request")
	}
}

func TestGraphQLBodyMatcherRejectsDifferentAliasArgumentAndInvalidQuery(t *testing.T) {
	matcher := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "{ mainHero: hero(episode: NEWHOPE) { name } }"
	  }
	}`)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "different alias",
			body: `{"query": "{ hero: hero(episode: NEWHOPE) { name } }"}`,
		},
		{
			name: "different argument",
			body: `{"query": "{ mainHero: hero(episode: JEDI) { name } }"}`,
		},
		{
			name: "invalid GraphQL",
			body: `{"query": "{ mainHero: hero(episode: ) { name } }"}`,
		},
		{
			name: "invalid JSON",
			body: `{"query":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if matcher.Matches(NewBodyContext([]byte(tt.body))) {
				t.Fatalf("GraphQL matcher matched %s, want no match", tt.name)
			}
		})
	}
}

func TestGraphQLBodyMatcherMatchesFragmentsAndSelectionOrder(t *testing.T) {
	matcher := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "fragment heroDetails on Hero { name age } query GetHero { hero { ...heroDetails id } }"
	  }
	}`)

	body := []byte(`{
	  "query": "query GetHero { hero { id ...heroDetails } } fragment heroDetails on Hero { age name }"
	}`)
	if !matcher.Matches(NewBodyContext(body)) {
		t.Fatalf("GraphQL matcher did not match equivalent fragment query")
	}
}

func TestGraphQLBodyMatcherVariablesRules(t *testing.T) {
	withVariables := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { name age } }",
	    "variables": {
	      "ids": [1, 2, 3]
	    }
	  }
	}`)
	if !withVariables.Matches(NewBodyContext([]byte(`{
	  "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { age name } }",
	  "variables": {
	    "ids": [1, 2, 3]
	  }
	}`))) {
		t.Fatalf("GraphQL matcher did not match equal variables")
	}
	if withVariables.Matches(NewBodyContext([]byte(`{
	  "query": "query GetCharacters($ids: [ID!]) { characters(ids: $ids) { age name } }",
	  "variables": {
	    "ids": [3, 2, 1]
	  }
	}`))) {
		t.Fatalf("GraphQL matcher matched variables with different array order")
	}

	withoutVariables := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "{ hero { name } }"
	  }
	}`)
	if !withoutVariables.Matches(NewBodyContext([]byte(`{"query":"{ hero { name } }"}`))) {
		t.Fatalf("GraphQL matcher without expected variables did not match absent request variables")
	}
	if withoutVariables.Matches(NewBodyContext([]byte(`{
	  "query": "{ hero { name } }",
	  "variables": {
	    "id": 1
	  }
	}`))) {
		t.Fatalf("GraphQL matcher matched unexpected request variables")
	}
}

func TestGraphQLBodyMatcherOperationNameRules(t *testing.T) {
	withOperation := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "query GetHero { hero { name } }",
	    "operationName": "GetHero"
	  }
	}`)
	if !withOperation.Matches(NewBodyContext([]byte(`{
	  "query": "query GetHero { hero { name } }",
	  "operationName": "GetHero"
	}`))) {
		t.Fatalf("GraphQL matcher did not match equal operationName")
	}
	if withOperation.Matches(NewBodyContext([]byte(`{
	  "query": "query GetHero { hero { name } }",
	  "operationName": "Other"
	}`))) {
		t.Fatalf("GraphQL matcher matched different operationName")
	}

	withoutOperation := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "{ hero { name } }"
	  }
	}`)
	if withoutOperation.Matches(NewBodyContext([]byte(`{
	  "query": "{ hero { name } }",
	  "operationName": "GetHero"
	}`))) {
		t.Fatalf("GraphQL matcher matched unexpected operationName")
	}
}

func TestParseCustomMatcherRejectsInvalidGraphQLMatcher(t *testing.T) {
	tests := []string{
		`{"name":"unknown","parameters":{"query":"{ hero { name } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero { } }"}}`,
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseCustomMatcher(json.RawMessage(body)); err == nil {
				t.Fatalf("ParseCustomMatcher() error = nil, want error")
			}
		})
	}
}

func mustParseCustomMatcher(t *testing.T, body string) *CustomMatcher {
	t.Helper()

	matcher, err := ParseCustomMatcher(json.RawMessage(body))
	if err != nil {
		t.Fatalf("ParseCustomMatcher(): %v", err)
	}
	return matcher
}
