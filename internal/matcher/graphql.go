package matcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const GraphQLBodyMatcherName = "graphql-body-matcher"

type CustomMatcher struct {
	graphql *GraphQLBodyMatcher
}

type GraphQLBodyMatcher struct {
	query        graphqlDocument
	hasVariables bool
	variables    []byte
	hasOperation bool
	operation    string
}

func ParseCustomMatcher(raw json.RawMessage) (*CustomMatcher, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("request.customMatcher must be a JSON object")
	}

	name, err := stringFromObject(object, "name", true)
	if err != nil {
		return nil, err
	}
	if name != GraphQLBodyMatcherName {
		return nil, fmt.Errorf("unsupported request.customMatcher.name %q", name)
	}

	graphQL, err := parseGraphQLBodyMatcher(object["parameters"])
	if err != nil {
		return nil, err
	}
	return &CustomMatcher{graphql: &graphQL}, nil
}

func (m *CustomMatcher) Matches(body *BodyContext) bool {
	if m == nil {
		return true
	}
	if m.graphql != nil {
		return m.graphql.Matches(body)
	}
	return true
}

func parseGraphQLBodyMatcher(raw json.RawMessage) (GraphQLBodyMatcher, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return GraphQLBodyMatcher{}, fmt.Errorf("request.customMatcher.parameters is required")
	}

	var parameters map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parameters); err != nil || parameters == nil {
		return GraphQLBodyMatcher{}, fmt.Errorf("request.customMatcher.parameters must be a JSON object")
	}

	query, err := stringFromObject(parameters, "query", true)
	if err != nil {
		return GraphQLBodyMatcher{}, err
	}
	document, err := parseGraphQLDocument(query)
	if err != nil {
		return GraphQLBodyMatcher{}, fmt.Errorf("request.customMatcher.parameters.query: %w", err)
	}

	matcher := GraphQLBodyMatcher{query: document}
	if rawVariables, ok := parameters["variables"]; ok && !isJSONNull(rawVariables) {
		normalized, err := normalizeRawJSON(rawVariables)
		if err != nil {
			return GraphQLBodyMatcher{}, fmt.Errorf("request.customMatcher.parameters.variables must be valid JSON: %w", err)
		}
		matcher.hasVariables = true
		matcher.variables = normalized
	}
	if rawOperationName, ok := parameters["operationName"]; ok && !isJSONNull(rawOperationName) {
		operationName, err := stringRaw(rawOperationName, "request.customMatcher.parameters.operationName")
		if err != nil {
			return GraphQLBodyMatcher{}, err
		}
		matcher.hasOperation = true
		matcher.operation = operationName
	}

	return matcher, nil
}

func (m GraphQLBodyMatcher) Matches(body *BodyContext) bool {
	parsed, err := parseGraphQLRequestBody(body.Raw())
	if err != nil {
		return false
	}

	requestDocument, err := parseGraphQLDocument(parsed.Query)
	if err != nil {
		return false
	}
	if m.query.canonical != requestDocument.canonical {
		return false
	}

	if m.hasVariables != parsed.HasVariables {
		return false
	}
	if m.hasVariables && !bytes.Equal(m.variables, parsed.Variables) {
		return false
	}

	if m.hasOperation != parsed.HasOperation {
		return false
	}
	if m.hasOperation && m.operation != parsed.Operation {
		return false
	}

	return true
}

type graphQLRequestBody struct {
	Query        string
	HasVariables bool
	Variables    []byte
	HasOperation bool
	Operation    string
}

func parseGraphQLRequestBody(raw []byte) (graphQLRequestBody, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return graphQLRequestBody{}, fmt.Errorf("GraphQL request body must be a JSON object")
	}

	query, err := stringFromObject(object, "query", true)
	if err != nil {
		return graphQLRequestBody{}, err
	}

	body := graphQLRequestBody{Query: query}
	if rawVariables, ok := object["variables"]; ok && !isJSONNull(rawVariables) {
		normalized, err := normalizeRawJSON(rawVariables)
		if err != nil {
			return graphQLRequestBody{}, err
		}
		body.HasVariables = true
		body.Variables = normalized
	}
	if rawOperationName, ok := object["operationName"]; ok && !isJSONNull(rawOperationName) {
		operationName, err := stringRaw(rawOperationName, "operationName")
		if err != nil {
			return graphQLRequestBody{}, err
		}
		body.HasOperation = true
		body.Operation = operationName
	}
	return body, nil
}

func stringFromObject(object map[string]json.RawMessage, field string, required bool) (string, error) {
	raw, ok := object[field]
	if !ok || len(raw) == 0 || isJSONNull(raw) {
		if required {
			return "", fmt.Errorf("%s is required", field)
		}
		return "", nil
	}
	return stringRaw(raw, field)
}

func stringRaw(raw json.RawMessage, field string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", field)
	}
	return value, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

type graphqlDocument struct {
	canonical string
}

func parseGraphQLDocument(query string) (graphqlDocument, error) {
	parser, err := newGraphQLParser(query)
	if err != nil {
		return graphqlDocument{}, err
	}
	canonical, err := parser.parseDocument()
	if err != nil {
		return graphqlDocument{}, err
	}
	return graphqlDocument{canonical: canonical}, nil
}

type graphQLParser struct {
	tokens []graphQLToken
	pos    int
}

type graphQLToken struct {
	kind  graphQLTokenKind
	value string
}

type graphQLTokenKind int

const (
	graphQLTokenName graphQLTokenKind = iota
	graphQLTokenString
	graphQLTokenNumber
	graphQLTokenPunct
)

func newGraphQLParser(query string) (*graphQLParser, error) {
	tokens, err := tokenizeGraphQL(query)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("GraphQL query must not be empty")
	}
	return &graphQLParser{tokens: tokens}, nil
}

func tokenizeGraphQL(query string) ([]graphQLToken, error) {
	var tokens []graphQLToken
	for i := 0; i < len(query); {
		r := rune(query[i])
		if r == ',' || unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '#' {
			for i < len(query) && query[i] != '\n' && query[i] != '\r' {
				i++
			}
			continue
		}
		if isGraphQLNameStart(r) {
			start := i
			i++
			for i < len(query) && isGraphQLNameContinue(rune(query[i])) {
				i++
			}
			tokens = append(tokens, graphQLToken{kind: graphQLTokenName, value: query[start:i]})
			continue
		}
		if r == '"' {
			value, next, err := readGraphQLString(query, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, graphQLToken{kind: graphQLTokenString, value: value})
			i = next
			continue
		}
		if r == '-' || isDigit(r) {
			value, next, err := readGraphQLNumber(query, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, graphQLToken{kind: graphQLTokenNumber, value: value})
			i = next
			continue
		}
		if strings.HasPrefix(query[i:], "...") {
			tokens = append(tokens, graphQLToken{kind: graphQLTokenPunct, value: "..."})
			i += 3
			continue
		}
		if strings.ContainsRune("!$():=@[]{}|", r) {
			tokens = append(tokens, graphQLToken{kind: graphQLTokenPunct, value: string(r)})
			i++
			continue
		}
		return nil, fmt.Errorf("unsupported GraphQL character %q", r)
	}
	return tokens, nil
}

func readGraphQLString(query string, start int) (string, int, error) {
	if strings.HasPrefix(query[start:], `"""`) {
		end := strings.Index(query[start+3:], `"""`)
		if end == -1 {
			return "", 0, fmt.Errorf("unterminated GraphQL block string")
		}
		return query[start+3 : start+3+end], start + 3 + end + 3, nil
	}

	for i := start + 1; i < len(query); i++ {
		if query[i] == '\\' {
			i++
			continue
		}
		if query[i] == '"' {
			unquoted, err := strconv.Unquote(query[start : i+1])
			if err != nil {
				return "", 0, fmt.Errorf("invalid GraphQL string: %w", err)
			}
			return unquoted, i + 1, nil
		}
	}
	return "", 0, fmt.Errorf("unterminated GraphQL string")
}

func readGraphQLNumber(query string, start int) (string, int, error) {
	i := start
	if query[i] == '-' {
		i++
	}
	if i >= len(query) || !isDigit(rune(query[i])) {
		return "", 0, fmt.Errorf("invalid GraphQL number")
	}
	for i < len(query) && isDigit(rune(query[i])) {
		i++
	}
	if i < len(query) && query[i] == '.' {
		i++
		if i >= len(query) || !isDigit(rune(query[i])) {
			return "", 0, fmt.Errorf("invalid GraphQL float")
		}
		for i < len(query) && isDigit(rune(query[i])) {
			i++
		}
	}
	if i < len(query) && (query[i] == 'e' || query[i] == 'E') {
		i++
		if i < len(query) && (query[i] == '+' || query[i] == '-') {
			i++
		}
		if i >= len(query) || !isDigit(rune(query[i])) {
			return "", 0, fmt.Errorf("invalid GraphQL exponent")
		}
		for i < len(query) && isDigit(rune(query[i])) {
			i++
		}
	}
	return query[start:i], i, nil
}

func isGraphQLNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isGraphQLNameContinue(r rune) bool {
	return isGraphQLNameStart(r) || isDigit(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func (p *graphQLParser) parseDocument() (string, error) {
	var definitions []string
	for !p.eof() {
		definition, err := p.parseDefinition()
		if err != nil {
			return "", err
		}
		definitions = append(definitions, definition)
	}
	sort.Strings(definitions)
	return "doc(" + strings.Join(definitions, "|") + ")", nil
}

func (p *graphQLParser) parseDefinition() (string, error) {
	switch {
	case p.peekName("fragment"):
		return p.parseFragmentDefinition()
	case p.peekName("query"), p.peekName("mutation"), p.peekName("subscription"):
		return p.parseOperationDefinition()
	case p.peekPunct("{"):
		selections, err := p.parseSelectionSet()
		if err != nil {
			return "", err
		}
		return "op:query:::[]:" + selections, nil
	default:
		return "", fmt.Errorf("expected GraphQL definition")
	}
}

func (p *graphQLParser) parseFragmentDefinition() (string, error) {
	if err := p.expectNameValue("fragment"); err != nil {
		return "", err
	}
	name, err := p.expectName()
	if err != nil {
		return "", err
	}
	if err := p.expectNameValue("on"); err != nil {
		return "", err
	}
	typeName, err := p.expectName()
	if err != nil {
		return "", err
	}
	directives, err := p.parseDirectives()
	if err != nil {
		return "", err
	}
	selections, err := p.parseSelectionSet()
	if err != nil {
		return "", err
	}
	return "fragment:" + name + ":on:" + typeName + ":" + directives + ":" + selections, nil
}

func (p *graphQLParser) parseOperationDefinition() (string, error) {
	operation, _ := p.consumeName()
	name := ""
	if p.peekKind(graphQLTokenName) && !p.peekName("on") {
		consumed, _ := p.consumeName()
		name = consumed
	}
	variables := "[]"
	if p.peekPunct("(") {
		parsed, err := p.parseVariableDefinitions()
		if err != nil {
			return "", err
		}
		variables = parsed
	}
	directives, err := p.parseDirectives()
	if err != nil {
		return "", err
	}
	selections, err := p.parseSelectionSet()
	if err != nil {
		return "", err
	}
	return "op:" + operation + ":" + name + ":" + variables + ":" + directives + ":" + selections, nil
}

func (p *graphQLParser) parseVariableDefinitions() (string, error) {
	if err := p.expectPunct("("); err != nil {
		return "", err
	}
	var definitions []string
	for !p.peekPunct(")") {
		if err := p.expectPunct("$"); err != nil {
			return "", err
		}
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		if err := p.expectPunct(":"); err != nil {
			return "", err
		}
		typeName, err := p.parseTypeRef()
		if err != nil {
			return "", err
		}
		defaultValue := ""
		if p.peekPunct("=") {
			p.pos++
			value, err := p.parseValue()
			if err != nil {
				return "", err
			}
			defaultValue = "=" + value
		}
		definitions = append(definitions, name+":"+typeName+defaultValue)
	}
	if err := p.expectPunct(")"); err != nil {
		return "", err
	}
	sort.Strings(definitions)
	return "[" + strings.Join(definitions, ",") + "]", nil
}

func (p *graphQLParser) parseTypeRef() (string, error) {
	var base string
	if p.peekPunct("[") {
		p.pos++
		inner, err := p.parseTypeRef()
		if err != nil {
			return "", err
		}
		if err := p.expectPunct("]"); err != nil {
			return "", err
		}
		base = "[" + inner + "]"
	} else {
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		base = name
	}
	if p.peekPunct("!") {
		p.pos++
		base += "!"
	}
	return base, nil
}

func (p *graphQLParser) parseSelectionSet() (string, error) {
	if err := p.expectPunct("{"); err != nil {
		return "", err
	}
	var selections []string
	for !p.peekPunct("}") {
		selection, err := p.parseSelection()
		if err != nil {
			return "", err
		}
		selections = append(selections, selection)
	}
	if err := p.expectPunct("}"); err != nil {
		return "", err
	}
	if len(selections) == 0 {
		return "", fmt.Errorf("GraphQL selection set must not be empty")
	}
	sort.Strings(selections)
	return "sel[" + strings.Join(selections, ",") + "]", nil
}

func (p *graphQLParser) parseSelection() (string, error) {
	if p.peekPunct("...") {
		p.pos++
		if p.peekName("on") {
			p.pos++
			typeName, err := p.expectName()
			if err != nil {
				return "", err
			}
			directives, err := p.parseDirectives()
			if err != nil {
				return "", err
			}
			selections, err := p.parseSelectionSet()
			if err != nil {
				return "", err
			}
			return "inline:on:" + typeName + ":" + directives + ":" + selections, nil
		}
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		directives, err := p.parseDirectives()
		if err != nil {
			return "", err
		}
		return "spread:" + name + ":" + directives, nil
	}
	return p.parseField()
}

func (p *graphQLParser) parseField() (string, error) {
	first, err := p.expectName()
	if err != nil {
		return "", err
	}
	alias := ""
	name := first
	if p.peekPunct(":") {
		p.pos++
		alias = first
		name, err = p.expectName()
		if err != nil {
			return "", err
		}
	}
	arguments, err := p.parseArguments()
	if err != nil {
		return "", err
	}
	directives, err := p.parseDirectives()
	if err != nil {
		return "", err
	}
	selections := ""
	if p.peekPunct("{") {
		selections, err = p.parseSelectionSet()
		if err != nil {
			return "", err
		}
	}
	return "field:" + alias + ":" + name + ":" + arguments + ":" + directives + ":" + selections, nil
}

func (p *graphQLParser) parseArguments() (string, error) {
	if !p.peekPunct("(") {
		return "args[]", nil
	}
	p.pos++
	var args []string
	for !p.peekPunct(")") {
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		if err := p.expectPunct(":"); err != nil {
			return "", err
		}
		value, err := p.parseValue()
		if err != nil {
			return "", err
		}
		args = append(args, name+":"+value)
	}
	if err := p.expectPunct(")"); err != nil {
		return "", err
	}
	sort.Strings(args)
	return "args[" + strings.Join(args, ",") + "]", nil
}

func (p *graphQLParser) parseDirectives() (string, error) {
	var directives []string
	for p.peekPunct("@") {
		p.pos++
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		arguments, err := p.parseArguments()
		if err != nil {
			return "", err
		}
		directives = append(directives, name+":"+arguments)
	}
	sort.Strings(directives)
	return "dirs[" + strings.Join(directives, ",") + "]", nil
}

func (p *graphQLParser) parseValue() (string, error) {
	if p.peekPunct("$") {
		p.pos++
		name, err := p.expectName()
		if err != nil {
			return "", err
		}
		return "$" + name, nil
	}
	if p.peekPunct("[") {
		p.pos++
		var values []string
		for !p.peekPunct("]") {
			value, err := p.parseValue()
			if err != nil {
				return "", err
			}
			values = append(values, value)
		}
		if err := p.expectPunct("]"); err != nil {
			return "", err
		}
		return "list[" + strings.Join(values, ",") + "]", nil
	}
	if p.peekPunct("{") {
		p.pos++
		var fields []string
		for !p.peekPunct("}") {
			name, err := p.expectName()
			if err != nil {
				return "", err
			}
			if err := p.expectPunct(":"); err != nil {
				return "", err
			}
			value, err := p.parseValue()
			if err != nil {
				return "", err
			}
			fields = append(fields, name+":"+value)
		}
		if err := p.expectPunct("}"); err != nil {
			return "", err
		}
		sort.Strings(fields)
		return "obj{" + strings.Join(fields, ",") + "}", nil
	}
	if token, ok := p.consumeKind(graphQLTokenString); ok {
		return "str:" + strconv.Quote(token.value), nil
	}
	if token, ok := p.consumeKind(graphQLTokenNumber); ok {
		return "num:" + token.value, nil
	}
	if token, ok := p.consumeKind(graphQLTokenName); ok {
		return "name:" + token.value, nil
	}
	return "", fmt.Errorf("expected GraphQL value")
}

func (p *graphQLParser) eof() bool {
	return p.pos >= len(p.tokens)
}

func (p *graphQLParser) peekKind(kind graphQLTokenKind) bool {
	return !p.eof() && p.tokens[p.pos].kind == kind
}

func (p *graphQLParser) peekName(value string) bool {
	return !p.eof() && p.tokens[p.pos].kind == graphQLTokenName && p.tokens[p.pos].value == value
}

func (p *graphQLParser) peekPunct(value string) bool {
	return !p.eof() && p.tokens[p.pos].kind == graphQLTokenPunct && p.tokens[p.pos].value == value
}

func (p *graphQLParser) consumeName() (string, bool) {
	token, ok := p.consumeKind(graphQLTokenName)
	if !ok {
		return "", false
	}
	return token.value, true
}

func (p *graphQLParser) consumeKind(kind graphQLTokenKind) (graphQLToken, bool) {
	if !p.peekKind(kind) {
		return graphQLToken{}, false
	}
	token := p.tokens[p.pos]
	p.pos++
	return token, true
}

func (p *graphQLParser) expectName() (string, error) {
	name, ok := p.consumeName()
	if !ok {
		return "", fmt.Errorf("expected GraphQL name")
	}
	return name, nil
}

func (p *graphQLParser) expectNameValue(value string) error {
	if !p.peekName(value) {
		return fmt.Errorf("expected GraphQL name %q", value)
	}
	p.pos++
	return nil
}

func (p *graphQLParser) expectPunct(value string) error {
	if !p.peekPunct(value) {
		return fmt.Errorf("expected GraphQL token %q", value)
	}
	p.pos++
	return nil
}
