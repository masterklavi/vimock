package matcher

import (
	"encoding/json"
	"testing"
)

func TestGraphQLParserCoversValuesDirectivesAndInlineFragments(t *testing.T) {
	matcher := mustParseCustomMatcher(t, `{
	  "name": "graphql-body-matcher",
	  "parameters": {
	    "query": "query Search($term: String!, $ids: [ID!]) @trace(enabled: true) { search(text: \"abc\", limit: 10, filter: {active: true, ids: [1, 2]}, ids: $ids) @include(if: true) { ... on Product @typed(name: \"p\") { id name } ...commonFields } } fragment commonFields on Node { createdAt }"
	  }
	}`)
	body := []byte(`{
	  "query": "fragment commonFields on Node { createdAt } query Search($ids:[ID!], $term:String!) @trace(enabled:true) { search(ids:$ids, filter:{ids:[1,2], active:true}, limit:10, text:\"abc\") @include(if:true) { ...commonFields ... on Product @typed(name:\"p\") { name id } } }"
	}`)
	if !matcher.Matches(NewBodyContext(body)) {
		t.Fatal("GraphQL matcher did not match equivalent query with directives and inline fragments")
	}
}

func TestGraphQLParserRejectsTokenizerAndParserErrors(t *testing.T) {
	tests := []string{
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero(name: \"unterminated) { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero(value: 1.2.3) { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero(value: @) { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"fragment on Hero { id }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"query Test($id ID) { hero { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"query Test($id: [ID!) { hero { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero(episode) { id } }"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ ... }"}}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseCustomMatcher(json.RawMessage(body)); err == nil {
				t.Fatal("ParseCustomMatcher() error = nil, want error")
			}
		})
	}
}

func TestParseCustomMatcherNullAndParameterValidation(t *testing.T) {
	matcher, err := ParseCustomMatcher(nil)
	if err != nil || matcher != nil {
		t.Fatalf("ParseCustomMatcher(nil) = %v, %v; want nil nil", matcher, err)
	}
	matcher, err = ParseCustomMatcher(json.RawMessage(`null`))
	if err != nil || matcher != nil {
		t.Fatalf("ParseCustomMatcher(null) = %v, %v; want nil nil", matcher, err)
	}
	invalid := []string{
		`[]`,
		`{"name": 1}`,
		`{"name":"graphql-body-matcher","parameters":[]}`,
		`{"name":"graphql-body-matcher","parameters":{"query":1}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero { id } }","variables":"bad"}}`,
		`{"name":"graphql-body-matcher","parameters":{"query":"{ hero { id } }","operationName":1}}`,
	}
	for _, body := range invalid {
		t.Run(body, func(t *testing.T) {
			if _, err := ParseCustomMatcher(json.RawMessage(body)); err == nil {
				t.Fatal("ParseCustomMatcher() error = nil, want error")
			}
		})
	}
}
