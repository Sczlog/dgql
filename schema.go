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

var objectTypeMap = make(map[string]*ObjectDefinition)

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
	if t.IsNonNull {
		return fmt.Sprintf("%s!", t.Name)
	}
	return t.Name
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

func (o ObjectDefinition) parseObjectOutput(nested bool) string {
	fields := make([]string, 0)
	for _, field := range o.Fields {
		switch field.Type.Kind {
		case "SCALAR":
			fallthrough
		case "ENUM":
			fields = append(fields, field.Name)
		case "OBJECT":
			if !nested {
				typeDef := objectTypeMap[field.Type.Name]
				if typeDef != nil {
					nestedQuery := typeDef.parseObjectOutput(true)
					fields = append(fields, fmt.Sprintf("%s %s ", field.Name, nestedQuery))
				} else {
					panic(fmt.Sprintf("Object %s not found", field.Type.Name))
				}
			}
		}
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, " "))
}

func (t IntrospectionTypeRef) parseOutputType() string {
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
	// for scalar type, no nest query is needed
	case "SCALAR":
		fallthrough
	case "ENUM":
		return ""
	case "OBJECT":
		typeDef := objectTypeMap[typeName.Name]
		if typeDef != nil {
			return typeDef.parseObjectOutput(false)
		} else {
			panic(fmt.Sprintf("Object %s not found", typeName.Name))
		}
	}
	panic(fmt.Sprintf("Unknown type %s", typeName.Name))
}

func (i *Introspection) ParseSchema() *GraphqlClient {
	var mutationDocumentMap = make(map[string]string)
	var queryDocumentMap = make(map[string]string)
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
		} else if t.Kind == "UNION" {
			//TODO: add support for union
		}
	}
	for _, query := range query.Fields {
		name := query.Name
		var argsStr string
		var resolverStr string
		if query.Args != nil && len(query.Args) > 0 {
			args := make([]string, len(query.Args))
			args2 := make([]string, len(query.Args))
			for idx, arg := range query.Args {
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
		} else {
			// resolver with no args
			argsStr = ""
			resolverStr = ""
		}
		output := query.Type.parseOutputType()
		queryDocumentMap[name] = fmt.Sprintf("query %s%s { %s%s %s}", name, argsStr, name, resolverStr, output)
	}
	for _, mutation := range mutation.Fields {
		name := mutation.Name
		var argsStr string
		var resolverStr string
		if mutation.Args != nil && len(mutation.Args) > 0 {
			args := make([]string, len(mutation.Args))
			args2 := make([]string, len(mutation.Args))
			for idx, arg := range mutation.Args {
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
		} else {
			// resolver with no args
			argsStr = ""
			resolverStr = ""
		}
		output := mutation.Type.parseOutputType()
		mutationDocumentMap[name] = fmt.Sprintf("mutation %s%s { %s%s %s}", name, argsStr, name, resolverStr, output)
	}
	return &GraphqlClient{
		queryDocumentMap:    queryDocumentMap,
		mutationDocumentMap: mutationDocumentMap,
		DefaultHeaders:      make(map[string]string),
		Endpoint:            i.Endpoint,
		Client:              resty.New(),
	}
}
