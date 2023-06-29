package dgql_test

import (
	"context"
	"testing"

	"github.com/Sczlog/dgql"
	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	client, err := dgql.NewClient("http://localhost:8080/graphql")
	pass := assert.Equal(t, nil, err, "Error creating client")
	if !pass {
		return
	}
	resp, _, err := client.Query(context.Background(), "product", map[string]interface{}{
		"id": 1,
	}, nil)
	pass = assert.Equal(t, nil, err, "Error querying")
	if !pass {
		return
	}
	println(resp.Raw)
}

func TestMutation(t *testing.T) {
	client, err := dgql.NewClient("http://localhost:8080/graphql")
	pass := assert.Equal(t, nil, err, "Error creating client")
	if !pass {
		return
	}
	resp, _, err := client.Mutation(context.Background(), "create", map[string]interface{}{
		"name":  "test",
		"info":  "test",
		"price": 100,
	}, nil)
	pass = assert.Equal(t, nil, err, "Error mutating")
	if !pass {
		return
	}
	println(resp.Raw)
}
