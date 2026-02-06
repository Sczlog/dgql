package dgql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type GraphqlClient struct {
	mutationDocumentMap map[string]string
	queryDocumentMap    map[string]string
	DefaultHeaders      map[string]string
	Endpoint            string
	Client              *resty.Client
}

func (c *GraphqlClient) Query(ctx context.Context, operationName string, variables interface{}, headers *map[string]string) (*gjson.Result, *http.Header, error) {
	document, ok := c.queryDocumentMap[operationName]
	if !ok {
		return nil, nil, fmt.Errorf("query operation %s not found", operationName)
	}
	return c.Raw(ctx, document, operationName, variables, headers)
}

func (c *GraphqlClient) Mutation(ctx context.Context, operationName string, variables interface{}, headers *map[string]string) (*gjson.Result, *http.Header, error) {
	document, ok := c.mutationDocumentMap[operationName]
	if !ok {
		return nil, nil, fmt.Errorf("mutation operation %s not found", operationName)
	}
	return c.Raw(ctx, document, operationName, variables, headers)
}

func (c *GraphqlClient) UploadMutation(ctx context.Context, operationName string, variables interface{}, headers *map[string]string, files []FileConfig) (*gjson.Result, *http.Header, error) {
	document, ok := c.mutationDocumentMap[operationName]
	if !ok {
		return nil, nil, fmt.Errorf("mutation operation %s not found", operationName)
	}
	return c.RawUpload(ctx, document, operationName, variables, headers, files)
}

func (c *GraphqlClient) Raw(ctx context.Context, document string, operationName string, variables interface{}, headers *map[string]string) (*gjson.Result, *http.Header, error) {
	request := c.Client.R()
	if ctx != nil {
		request.SetContext(ctx)
	}
	for k, v := range c.DefaultHeaders {
		request.SetHeader(k, v)
	}
	if headers != nil {
		for k, v := range *headers {
			request.SetHeader(k, v)
		}
	}

	resp, err := request.
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"query":         document,
			"operationName": operationName,
			"variables":     variables,
		}).Post(c.Endpoint)
	if err != nil {
		return nil, nil, err
	}
	result := gjson.ParseBytes(resp.Body())
	gqlerror := result.Get("errors")
	if gqlerror.Exists() {
		return nil, nil, fmt.Errorf("%v", gqlerror)
	}
	gqldata := result.Get("data")
	if !gqldata.Exists() {
		return nil, nil, fmt.Errorf("data not found")
	}
	respHeader := resp.Header()
	return &gqldata, &respHeader, nil
}

type FileConfig struct {
	Bytes *[]byte
	Path  string
}

type uploadOperations struct {
	Query         string      `json:"query"`
	OperationName string      `json:"operationName"`
	Variables     interface{} `json:"variables"`
}

func (c *GraphqlClient) RawUpload(ctx context.Context, document string, operationName string, variables interface{}, headers *map[string]string, files []FileConfig) (*gjson.Result, *http.Header, error) {
	request := c.Client.R()
	if ctx != nil {
		request.SetContext(ctx)
	}
	for k, v := range c.DefaultHeaders {
		request.SetHeader(k, v)
	}
	if headers != nil {
		for k, v := range *headers {
			request.SetHeader(k, v)
		}
	}
	// as httpclient use map[string][]string as formdata, which make formdata's order unreliable
	// use mime package here to build raw body
	var bBody bytes.Buffer
	writer := multipart.NewWriter(&bBody)
	mapping := make(map[string][]string)
	for i, file := range files {
		mapping[fmt.Sprintf("%d", i)] = []string{fmt.Sprintf("variables.%s", file.Path)}
	}
	bMapping, err := json.Marshal(mapping)
	if err != nil {
		return nil, nil, err
	}
	operations, err := json.Marshal(uploadOperations{
		Query:         document,
		OperationName: operationName,
		Variables:     variables,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := writer.WriteField("operations", string(operations)); err != nil {
		return nil, nil, err
	}
	if err := writer.WriteField("map", string(bMapping)); err != nil {
		return nil, nil, err
	}
	for i, file := range files {
		part, err := writer.CreateFormFile(fmt.Sprintf("%d", i), "file")
		if err != nil {
			return nil, nil, err
		}
		if _, err = part.Write(*file.Bytes); err != nil {
			return nil, nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}
	request.SetBody(bBody.Bytes())
	request.SetHeader("Content-Type", writer.FormDataContentType())

	resp, err := request.Post(c.Endpoint)
	if err != nil {
		return nil, nil, err
	}
	result := gjson.ParseBytes(resp.Body())
	gqlerror := result.Get("errors")
	if gqlerror.Exists() {
		return nil, nil, fmt.Errorf("%v", gqlerror)
	}
	gqldata := result.Get("data")
	if !gqldata.Exists() {
		return nil, nil, fmt.Errorf("data not found")
	}
	respHeaders := resp.Header()
	return &gqldata, &respHeaders, nil
}

func NewClient(endpoint string) (*GraphqlClient, error) {
	introspection, err := getIntrospection(endpoint)
	if err != nil {
		return nil, err
	}
	client, err := introspection.ParseSchema()
	if err != nil {
		return nil, err
	}
	return client, nil
}
