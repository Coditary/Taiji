package rag

// Hit is one retrieved chunk.
type Hit struct {
	Scope    string
	ID       string
	Score    float32
	Text     string
	Source   string
	ChunkIdx int
}

// IndexRequest ingests documents into a collection.
type IndexRequest struct {
	Collection   string
	SourcePaths  []string
	Recursive    bool
	Glob         string
	Exclude      []string
	StoreDir     string
	ChunkSize    int
	EmbedModel   string
	IndexMode    string
}

// QueryRequest retrieves relevant chunks.
type QueryRequest struct {
	Collection string
	Query      string
	TopK       int
	MinScore   float32
	StoreDir   string
}

// IndexResult summarizes indexing.
type IndexResult struct {
	Scope      string
	Collection string
	Message    string
	ChunksAdded int
	TotalChunks int
}
