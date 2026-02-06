package dgql

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type RetrieveType struct {
	Name      string
	Kind      string
	IsList    bool
	IsNonNull bool
}

func (t IntrospectionOfType) retrieveType(parent *RetrieveType) *RetrieveType {
	if parent == nil {
		parent = &RetrieveType{}
	}
	if t.Kind == "LIST" {
		parent.IsList = true
		return t.OfType.retrieveType(parent)
	} else if t.Kind == "NON_NULL" {
		parent.IsNonNull = true
		return t.OfType.retrieveType(parent)
	}
	parent.Name = t.Name
	parent.Kind = t.Kind
	return parent
}

func (t RetrieveType) toArgString() string {
	typeName := t.Name
	if t.IsList {
		typeName = fmt.Sprintf("[%s]", typeName)
	}
	if t.IsNonNull {
		return fmt.Sprintf("%s!", typeName)
	}
	return typeName
}

type ObjectDefinition struct {
	Name   string
	Kind   string
	Fields []*ObjectFieldDefinition
}

type ObjectFieldDefinition struct {
	Name string
	Type *RetrieveType
}

func (t IntrospectionType) parseObject() *ObjectDefinition {
	var result = ObjectDefinition{
		Name: t.Name,
		Kind: t.Kind,
	}
	if t.Fields != nil && len(t.Fields) > 0 {
		var fields = make([]*ObjectFieldDefinition, 0)
		for _, field := range t.Fields {
			// ignore pageInfo and edge for connection response.
			if (field.Name == "pageInfo" || field.Name == "edges") && strings.HasSuffix(t.Name, "Connection") {
				continue
			}
			if field.Type != nil {
				var typeName *RetrieveType
				if field.Type.OfType != nil {
					typeName = &RetrieveType{
						IsList:    field.Type.Kind == "LIST",
						IsNonNull: field.Type.Kind == "NON_NULL",
					}
					typeName = field.Type.OfType.retrieveType(typeName)
				} else {
					typeName = &RetrieveType{
						Name: field.Type.Name,
						Kind: field.Type.Kind,
					}
				}
				fields = append(fields, &ObjectFieldDefinition{
					Name: field.Name,
					Type: typeName,
				})
			}
		}
		result.Fields = fields
	}
	return &result
}

func (o ObjectDefinition) parseObjectOutput(nested bool, objectTypeMap map[string]*ObjectDefinition) (string, error) {
	fields := make([]string, 0)
	for _, field := range o.Fields {
		switch field.Type.Kind {
		case "SCALAR", "ENUM":
			fields = append(fields, field.Name)
		case "OBJECT":
			if !nested {
				typeDef := objectTypeMap[field.Type.Name]
				if typeDef == nil {
					return "", fmt.Errorf("object %s not found", field.Type.Name)
				}
				nestedQuery, err := typeDef.parseObjectOutput(true, objectTypeMap)
				if err != nil {
					return "", err
				}
				fields = append(fields, fmt.Sprintf("%s %s ", field.Name, nestedQuery))
			}
		}
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, " ")), nil
}

func (t IntrospectionTypeRef) parseOutputType(objectTypeMap map[string]*ObjectDefinition) (string, error) {
	var typeName *RetrieveType
	if t.OfType != nil {
		typeName = &RetrieveType{
			IsList:    t.Kind == "LIST",
			IsNonNull: t.Kind == "NON_NULL",
		}
		typeName = t.OfType.retrieveType(typeName)
	} else {
		typeName = &RetrieveType{
			Name: t.Name,
			Kind: t.Kind,
		}
	}
	switch typeName.Kind {
	case "SCALAR", "ENUM":
		return "", nil
	case "OBJECT":
		typeDef := objectTypeMap[typeName.Name]
		if typeDef == nil {
			return "", fmt.Errorf("object %s not found", typeName.Name)
		}
		return typeDef.parseObjectOutput(false, objectTypeMap)
	}
	return "", fmt.Errorf("unknown type %s", typeName.Name)
}

func (i *Introspection) ParseSchema() (*GraphqlClient, error) {
	var mutationDocumentMap = make(map[string]string)
	var queryDocumentMap = make(map[string]string)
	var objectTypeMap = make(map[string]*ObjectDefinition)
	var query *IntrospectionType
	var mutation *IntrospectionType
	for _, t := range i.Schema.Types {
		if t.Kind == "OBJECT" {
			if t.Name == "Query" {
				query = t
			} else if t.Name == "Mutation" {
				mutation = t
			} else {
				objectTypeMap[t.Name] = t.parseObject()
			}
		}
	}
	if query != nil {
		for _, queryField := range query.Fields {
			name := queryField.Name
			argsStr := ""
			resolverStr := ""
			if queryField.Args != nil && len(queryField.Args) > 0 {
				args := make([]string, len(queryField.Args))
				args2 := make([]string, len(queryField.Args))
				for idx, arg := range queryField.Args {
					var typeName *RetrieveType
					if arg.Type.OfType != nil {
						typeName = &RetrieveType{
							IsList:    arg.Type.Kind == "LIST",
							IsNonNull: arg.Type.Kind == "NON_NULL",
						}
						typeName = arg.Type.OfType.retrieveType(typeName)
					} else {
						typeName = &RetrieveType{
							Name: arg.Type.Name,
							Kind: arg.Type.Kind,
						}
					}
					args[idx] = fmt.Sprintf("$%s: %s", arg.Name, typeName.toArgString())
					args2[idx] = fmt.Sprintf("%s: $%s", arg.Name, arg.Name)
				}
				argsStr = fmt.Sprintf("(%s)", strings.Join(args, ", "))
				resolverStr = fmt.Sprintf("(%s)", strings.Join(args2, ", "))
			}
			output, err := queryField.Type.parseOutputType(objectTypeMap)
			if err != nil {
				return nil, err
			}
			queryDocumentMap[name] = fmt.Sprintf("query %s%s { %s%s %s}", name, argsStr, name, resolverStr, output)
		}
	}
	if mutation != nil {
		for _, mutationField := range mutation.Fields {
			name := mutationField.Name
			argsStr := ""
			resolverStr := ""
			if mutationField.Args != nil && len(mutationField.Args) > 0 {
				args := make([]string, len(mutationField.Args))
				args2 := make([]string, len(mutationField.Args))
				for idx, arg := range mutationField.Args {
					var typeName *RetrieveType
					if arg.Type.OfType != nil {
						typeName = &RetrieveType{
							IsList:    arg.Type.Kind == "LIST",
							IsNonNull: arg.Type.Kind == "NON_NULL",
						}
						typeName = arg.Type.OfType.retrieveType(typeName)
					} else {
						typeName = &RetrieveType{
							Name: arg.Type.Name,
							Kind: arg.Type.Kind,
						}
					}
					args[idx] = fmt.Sprintf("$%s: %s", arg.Name, typeName.toArgString())
					args2[idx] = fmt.Sprintf("%s: $%s", arg.Name, arg.Name)
				}
				argsStr = fmt.Sprintf("(%s)", strings.Join(args, ", "))
				resolverStr = fmt.Sprintf("(%s)", strings.Join(args2, ", "))
			}
			output, err := mutationField.Type.parseOutputType(objectTypeMap)
			if err != nil {
				return nil, err
			}
			mutationDocumentMap[name] = fmt.Sprintf("mutation %s%s { %s%s %s}", name, argsStr, name, resolverStr, output)
		}
	}
	return &GraphqlClient{
		queryDocumentMap:    queryDocumentMap,
		mutationDocumentMap: mutationDocumentMap,
		DefaultHeaders:      make(map[string]string),
		Endpoint:            i.Endpoint,
		Client:              resty.New(),
	}, nil
}
