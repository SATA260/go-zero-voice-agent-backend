package ragclient

import "errors"

// ErrMissingUserID indicates the caller did not provide a required user identifier.
var ErrMissingUserID = errors.New("ragclient: missing user id")

// QueryRequest represents the payload accepted by the Python RAG service /query endpoint.
type QueryRequest struct {
	Query    string  `json:"query"`
	FileID   string  `json:"file_id,omitempty"`
	TopK     int     `json:"top_k"`
	EntityID *string `json:"entity_id,omitempty"`
}

// QueryMultipleRequest is sent to the /query-multiple endpoint for cross-file retrieval.
type QueryMultipleRequest struct {
	Query   string   `json:"query"`
	FileIDs []string `json:"file_ids"`
	TopK    int      `json:"top_k"`
}

// RetrievalResult captures a single vector similarity hit returned by the RAG service.
type RetrievalResult struct {
	PageContent string         `json:"page_content"`
	Metadata    map[string]any `json:"metadata"`
	Score       float64        `json:"score"`
}

// QueryResponse wraps the list of retrieval results produced by /query or /query-multiple.
type QueryResponse struct {
	Results []RetrievalResult `json:"results"`
}

// DocumentRecord represents a persisted chunk that can be fetched by id.
type DocumentRecord struct {
	PageContent string         `json:"page_content"`
	Metadata    map[string]any `json:"metadata"`
}

// ChunkRecord represents an item returned by /chunks pagination API.
type ChunkRecord struct {
	CustomID    string         `json:"custom_id"`
	PageContent string         `json:"page_content"`
	Metadata    map[string]any `json:"metadata"`
}

// DocumentsResponse is the payload returned by GET /documents.
type DocumentsResponse struct {
	Documents []DocumentRecord `json:"documents"`
}

// ListChunksResponse captures paginated chunk results.
type ListChunksResponse struct {
	Items    []ChunkRecord `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// ListChunksParams specifies filters for the chunk pagination API.
type ListChunksParams struct {
	Page     int
	PageSize int
	FileID   string
	EntityID string
	OrderBy  string
	Sort     string
}

// DeleteDocumentsResponse contains the outcome of deleting stored chunks.
type DeleteDocumentsResponse struct {
	DeletedCount int `json:"deleted_count"`
}

// EmbedCleanupMethod mirrors the Python CleanupMethod enum.
type EmbedCleanupMethod string

const (
	// EmbedCleanupIncremental keeps newly generated chunks without removing unchanged ones.
	EmbedCleanupIncremental EmbedCleanupMethod = "incremental"
	// EmbedCleanupFull removes all existing chunks before embedding the new file.
	EmbedCleanupFull EmbedCleanupMethod = "full"
)

// EmbedRequest contains the information required by /embed to fetch data from MinIO and index it.
type EmbedRequest struct {
	FileID        string
	BucketName    string
	ObjectPath    string
	Filename      string
	ContentType   string
	EntityID      string
	CleanupMethod EmbedCleanupMethod
}

// EmbedResponse holds the statistics reported after an embedding run.
type EmbedResponse struct {
	FileID         string   `json:"file_id"`
	EmbeddedChunks int      `json:"embedded_chunks"`
	SkippedChunks  int      `json:"skipped_chunks"`
	VectorIDs      []string `json:"vector_ids,omitempty"`
	Message        string   `json:"message,omitempty"`
}

// HealthResponse represents the service status returned by GET /health.
type HealthResponse struct {
	Status   string `json:"status"`
	Postgres bool   `json:"postgres"`
}
