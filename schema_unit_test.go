package dgql

import (
	"context"
	"strings"
	"testing"
)

func TestParseSchemaWithoutMutation(t *testing.T) {
	intro := &Introspection{
		Endpoint: "http://example.com/graphql",
		Schema: &IntrospectionSchema{
			Types: []*IntrospectionType{
				{Name: "String", Kind: "SCALAR"},
				{
					Name: "Query",
					Kind: "OBJECT",
					Fields: []*IntrospectionField{{
						Name: "ping",
						Type: &IntrospectionTypeRef{Kind: "SCALAR", Name: "String"},
					}},
				},
			},
		},
	}

	client, err := intro.ParseSchema()
	if err != nil {
		t.Fatalf("ParseSchema returned error: %v", err)
	}
	if _, ok := client.queryDocumentMap["ping"]; !ok {
		t.Fatalf("expected ping query document to be generated")
	}
}

func TestQueryAndMutationMissingOperation(t *testing.T) {
	client := &GraphqlClient{
		queryDocumentMap:    map[string]string{},
		mutationDocumentMap: map[string]string{},
	}

	_, _, err := client.Query(context.Background(), "missingQuery", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing query error, got %v", err)
	}

	_, _, err = client.Mutation(context.Background(), "missingMutation", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing mutation error, got %v", err)
	}
}

func TestRetrieveTypeToArgStringListAndNonNull(t *testing.T) {
	r := RetrieveType{Name: "String", IsList: true, IsNonNull: true}
	if r.toArgString() != "[String]!" {
		t.Fatalf("unexpected arg string: %s", r.toArgString())
	}
}
