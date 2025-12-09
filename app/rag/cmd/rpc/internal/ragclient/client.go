package ragclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const userIDHeader = "X-User-ID"

// Client 用于与 Python RAG FastAPI 服务交互。
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// Option 用于构造 Client 时扩展可选配置。
type Option func(*options)

type options struct {
	httpClient *http.Client
	timeout    time.Duration
	hasTimeout bool
}

// WithHTTPClient 允许复用已有的 http.Client。
func WithHTTPClient(httpClient *http.Client) Option {
	return func(o *options) {
		o.httpClient = httpClient
	}
}

// WithTimeout 设置客户端的请求超时时间。
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.timeout = timeout
			o.hasTimeout = true
		}
	}
}

// NewClient 创建指向 FastAPI 服务的新客户端实例。
func NewClient(endpoint string, opts ...Option) (*Client, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return nil, errors.New("ragclient: endpoint must not be empty")
	}

	// 自动补全协议前缀，修复 "unsupported protocol scheme" 错误
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "http://" + trimmed
	}

	if _, err := url.Parse(trimmed); err != nil {
		return nil, fmt.Errorf("ragclient: invalid endpoint: %w", err)
	}

	// 应用调用方传入的可选配置（复用 http.Client、覆盖超时等）。
	options := options{}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	// 默认超时避免调用方未设置 http.Client 时出现无限等待。
	if options.hasTimeout {
		httpClient.Timeout = options.timeout
	} else if httpClient.Timeout == 0 {
		httpClient.Timeout = 30 * time.Second
	}

	client := &Client{
		endpoint:   strings.TrimRight(trimmed, "/"),
		httpClient: httpClient,
	}

	return client, nil
}

// Health 查询 FastAPI 服务的健康状态。
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.invoke(ctx, http.MethodGet, "/health", nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FetchDocuments 按 ID 获取已存储的文本切片。
func (c *Client) FetchDocuments(ctx context.Context, userID string, ids []string) (*DocumentsResponse, error) {
	if len(ids) == 0 {
		return nil, errors.New("ragclient: ids must not be empty")
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	// 服务端要求重复的 ids 查询参数，例如 /documents?ids=foo&ids=bar。
	query := url.Values{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		query.Add("ids", id)
	}

	if len(query) == 0 {
		return nil, errors.New("ragclient: no valid ids supplied")
	}

	headers := map[string]string{userIDHeader: userID}
	var resp DocumentsResponse
	if err := c.invoke(ctx, http.MethodGet, "/documents?"+query.Encode(), headers, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteDocuments 根据 ID 删除向量库中的切片。
func (c *Client) DeleteDocuments(ctx context.Context, userID string, ids []string) (*DeleteDocumentsResponse, error) {
	if len(ids) == 0 {
		return nil, errors.New("ragclient: ids must not be empty")
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	payload := ids
	headers := map[string]string{userIDHeader: userID, "Content-Type": "application/json"}
	var resp DeleteDocumentsResponse
	if err := c.invoke(ctx, http.MethodDelete, "/documents", headers, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListChunks 分页查询已写入向量库的文本切片。
func (c *Client) ListChunks(ctx context.Context, userID string, params *ListChunksParams) (*ListChunksResponse, error) {
	if userID == "" {
		return nil, ErrMissingUserID
	}

	cfg := ListChunksParams{}
	if params != nil {
		cfg = *params
	}

	page := cfg.Page
	if page <= 0 {
		page = 1
	}

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	orderBy := strings.TrimSpace(cfg.OrderBy)
	if orderBy == "" {
		orderBy = "chunk_index"
	}

	sort := strings.ToLower(strings.TrimSpace(cfg.Sort))
	if sort != "desc" {
		sort = "asc"
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("page_size", strconv.Itoa(pageSize))
	query.Set("order_by", orderBy)
	query.Set("sort", sort)

	if fileID := strings.TrimSpace(cfg.FileID); fileID != "" {
		query.Set("file_id", fileID)
	}
	if entityID := strings.TrimSpace(cfg.EntityID); entityID != "" {
		query.Set("entity_id", entityID)
	}

	headers := map[string]string{userIDHeader: userID}
	var resp ListChunksResponse
	if err := c.invoke(ctx, http.MethodGet, "/chunks?"+query.Encode(), headers, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Query 在单个文件范围内执行语义检索。
func (c *Client) Query(ctx context.Context, userID string, req *QueryRequest) (*QueryResponse, error) {
	if req == nil {
		return nil, errors.New("ragclient: query request must not be nil")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("ragclient: query text must not be empty")
	}
	if req.TopK <= 0 {
		req.TopK = 4
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	headers := map[string]string{userIDHeader: userID, "Content-Type": "application/json"}
	var resp QueryResponse
	if err := c.invoke(ctx, http.MethodPost, "/query", headers, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryMultiple 在多个文件范围内执行语义检索。
func (c *Client) QueryMultiple(ctx context.Context, userID string, req *QueryMultipleRequest) (*QueryResponse, error) {
	if req == nil {
		return nil, errors.New("ragclient: query-multiple request must not be nil")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("ragclient: query text must not be empty")
	}
	if len(req.FileIDs) == 0 {
		return nil, errors.New("ragclient: file ids must not be empty")
	}
	if req.TopK <= 0 {
		req.TopK = 4
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	headers := map[string]string{userIDHeader: userID, "Content-Type": "application/json"}
	var resp QueryResponse
	if err := c.invoke(ctx, http.MethodPost, "/query-multiple", headers, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Embed 触发对已存储于 MinIO 的文件进行切片并写入向量库。
func (c *Client) Embed(ctx context.Context, userID string, req *EmbedRequest) (*EmbedResponse, error) {
	if req == nil {
		return nil, errors.New("ragclient: embed request must not be nil")
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}
	if strings.TrimSpace(req.FileID) == "" || strings.TrimSpace(req.BucketName) == "" || strings.TrimSpace(req.ObjectPath) == "" {
		return nil, errors.New("ragclient: file_id, bucket_name and object_path must not be empty")
	}

	cleanup := req.CleanupMethod
	if cleanup == "" {
		cleanup = EmbedCleanupIncremental
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fields := map[string]string{
		"file_id":        req.FileID,
		"bucket_name":    req.BucketName,
		"file_path":      req.ObjectPath,
		"cleanup_method": string(cleanup),
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("ragclient: write field %s failed: %w", key, err)
		}
	}
	if strings.TrimSpace(req.Filename) != "" {
		if err := writer.WriteField("filename", req.Filename); err != nil {
			return nil, fmt.Errorf("ragclient: write filename failed: %w", err)
		}
	}
	if strings.TrimSpace(req.ContentType) != "" {
		if err := writer.WriteField("file_content_type", req.ContentType); err != nil {
			return nil, fmt.Errorf("ragclient: write content type failed: %w", err)
		}
	}
	if strings.TrimSpace(req.EntityID) != "" {
		if err := writer.WriteField("entity_id", req.EntityID); err != nil {
			return nil, fmt.Errorf("ragclient: write entity id failed: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("ragclient: close multipart writer failed: %w", err)
	}

	headers := map[string]string{userIDHeader: userID, "Content-Type": writer.FormDataContentType()}
	var resp EmbedResponse
	if err := c.invoke(ctx, http.MethodPost, "/embed", headers, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) invoke(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	body any,
	out any,
) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 将不同类型的请求体统一转换为 http.Client 可发送的形式。
	var bodyReader io.Reader
	switch v := body.(type) {
	case nil:
		bodyReader = nil
	case *bytes.Buffer:
		bodyReader = bytes.NewReader(v.Bytes())
	case io.Reader:
		bodyReader = v
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("ragclient: marshal request failed: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
		if headers == nil {
			headers = map[string]string{}
		}
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bodyReader)
	if err != nil {
		return fmt.Errorf("ragclient: build request failed: %w", err)
	}

	for k, v := range headers {
		if v == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	// 使用配置好的 http.Client 发送请求，并确保关闭响应体。
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ragclient: request failed: %w", err)
	}
	defer resp.Body.Close()

	// 将 FastAPI 返回的错误状态透传出来，便于排查问题。
	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("ragclient: %s %s returned %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("ragclient: decode response failed: %w", err)
	}

	return nil
}
