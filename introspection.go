package dgql

import (
	"encoding/json"
	"errors"

	"github.com/go-resty/resty/v2"
)

// introspection query
var introspectionQuery = `
  query IntrospectionQuery {
    __schema {
      types {
        ...FullType
      }
      directives {
        name
        description
		    locations
        args {
          ...InputValue
        }
      }
    }
  }

  fragment FullType on __Type {
    kind
    name
    description
    fields(includeDeprecated: true) {
      name
      description
      args {
        ...InputValue
      }
      type {
        ...TypeRef
      }
      isDeprecated
      deprecationReason
    }
    inputFields {
      ...InputValue
    }
    interfaces {
      ...TypeRef
    }
    enumValues(includeDeprecated: true) {
      name
      description
      isDeprecated
      deprecationReason
    }
    possibleTypes {
      ...TypeRef
    }
  }

  fragment InputValue on __InputValue {
    name
    description
    type { ...TypeRef }
    defaultValue
  }

  fragment TypeRef on __Type {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
        }
      }
    }
  }
`

type IntrospectionOfType struct {
	Kind   string               `json:"kind"`
	Name   string               `json:"name"`
	OfType *IntrospectionOfType `json:"ofType,omitempty"`
}

type IntrospectionType struct {
	Kind          string                     `json:"kind"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	Fields        []*IntrospectionField      `json:"fields"`
	InputFields   []*IntrospectionInputValue `json:"inputFields"`
	Interfaces    []*IntrospectionTypeRef    `json:"interfaces"`
	EnumValues    []*IntrospectionEnumValue  `json:"enumValues"`
	PossibleTypes []*IntrospectionTypeRef    `json:"possibleTypes"`
	OfType        *IntrospectionOfType       `json:"ofType"`
}

type IntrospectionTypeRef struct {
	Kind   string               `json:"kind"`
	Name   string               `json:"name"`
	OfType *IntrospectionOfType `json:"ofType"`
}

type IntrospectionField struct {
	Name              string                     `json:"name"`
	Description       string                     `json:"description"`
	Args              []*IntrospectionInputValue `json:"args"`
	Type              *IntrospectionTypeRef      `json:"type"`
	IsDeprecated      bool                       `json:"isDeprecated"`
	DeprecationReason string                     `json:"deprecationReason"`
}

type IntrospectionInputValue struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Type         *IntrospectionTypeRef `json:"type"`
	DefaultValue string                `json:"defaultValue"`
}

type IntrospectionEnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

type IntrospectionDirective struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Locations   []string                   `json:"locations"`
	Args        []*IntrospectionInputValue `json:"args"`
}

type IntrospectionSchema struct {
	Types      []*IntrospectionType      `json:"types"`
	Directives []*IntrospectionDirective `json:"directives"`
}

type IntrospectionQueryData struct {
	Schema *IntrospectionSchema `json:"__schema"`
}
type IntrospectionQuery struct {
	Data *IntrospectionQueryData `json:"data"`
}

type Introspection struct {
	Schema   *IntrospectionSchema
	Endpoint string
}

func getIntrospection(endpoint string) (*Introspection, error) {
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"query": introspectionQuery,
		}).
		Post(endpoint)
	if err != nil {
		return nil, err
	}
	var result IntrospectionQuery
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return nil, err
	}
	if result.Data == nil {
		return nil, errors.New("invaild response")
	}
	return &Introspection{
		Schema:   result.Data.Schema,
		Endpoint: endpoint,
	}, nil
}
