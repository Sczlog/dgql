# Dynamic Graphql Client Golang

Automaticly build graphql document from introspection query, query graphql like restful api.

Use operation name and variable to query graphql server, does not need document.

Use [gjson](https://github.com/tidwall/gjson) to handle graphql response.

Currently under development, lack of test and some feature.

### Features
1. Automaticly build graphql document from introspection query
2. support query, mutation and uploadMutation

### Quick start

run a simple graphql server

```sh
go run example/server.go
```

run a query
```go
package main

import (
    "context"
    "log"
    "github.com/Sczlog/dgql"
)

func main(){
	client, err := dgql.NewClient("http://localhost:8080/graphql")
    if err != nil {
        log.Fatal(err)
        return
    }
	resp, err := client.Query(context.Background(), "product", map[string]interface{}{
		"id": 1,
	}, nil)
    if err != nil {
        log.Fatal(err)
        return
    }
	println(resp.Raw)
    // response {"product":{"id":1,"info":"Chicha morada is a beverage originated in the Andean regions of Perú but is actually consumed at a national level (wiki)","name":"Chicha Morada","price":7.99}}
}
```
or mutation
```go
package main

import (
	"context"
	"log"

	"github.com/Sczlog/dgql"
)

func main() {
	client, err := dgql.NewClient("http://localhost:8080/graphql")
	if err != nil {
		log.Fatal(err)
		return
	}
	resp, err := client.Mutation(context.Background(), "create", map[string]interface{}{
		"price": 100,
		"name":  "test",
		"info":  "test",
	}, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	println(resp.Raw)
	// response {"create":{"id":42007,"info":"test","name":"test","price":100}}
}

```

### Roadmap

##### v0.1.0
- [x] query
- [x] mutation
- [x] uploadMutation

##### next
- [ ] filter operation
- [ ] directive
- [ ] subscription