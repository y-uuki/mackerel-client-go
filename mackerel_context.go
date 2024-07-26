package mackerel

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func requestGetContext[T any](ctx context.Context, client *Client, path string) (*T, error) {
	return requestNoBodyContext[T](ctx, client, http.MethodGet, path, nil)
}

func requestGetWithParamsContext[T any](ctx context.Context, client *Client, path string, params url.Values) (*T, error) {
	return requestNoBodyContext[T](ctx, client, http.MethodGet, path, params)
}

func requestGetAndReturnHeaderContext[T any](ctx context.Context, client *Client, path string) (*T, http.Header, error) {
	return requestInternalContext[T](ctx, client, http.MethodGet, path, nil, nil)
}

func requestPostContext[T any](ctx context.Context, client *Client, path string, payload any) (*T, error) {
	return requestJSONContext[T](ctx, client, http.MethodPost, path, payload)
}

func requestPutContext[T any](ctx context.Context, client *Client, path string, payload any) (*T, error) {
	return requestJSONContext[T](ctx, client, http.MethodPut, path, payload)
}

func requestDeleteContext[T any](ctx context.Context, client *Client, path string) (*T, error) {
	return requestNoBodyContext[T](ctx, client, http.MethodDelete, path, nil)
}

func requestJSONContext[T any](ctx context.Context, client *Client, method, path string, payload any) (*T, error) {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(payload)
	if err != nil {
		return nil, err
	}
	data, _, err := requestInternalContext[T](ctx, client, method, path, nil, &body)
	return data, err
}

func requestNoBodyContext[T any](ctx context.Context, client *Client, method, path string, params url.Values) (*T, error) {
	data, _, err := requestInternalContext[T](context.Background(), client, method, path, params, nil)
	return data, err
}

func requestInternalContext[T any](ctx context.Context, client *Client, method, path string, params url.Values, body io.Reader) (*T, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, method, client.urlFor(path, params).String(), body)
	if err != nil {
		return nil, nil, err
	}
	if body != nil || method != http.MethodGet {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := client.Request(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) // nolint
		resp.Body.Close()
	}()

	var data T
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, nil, err
	}
	return &data, resp.Header, nil
}
