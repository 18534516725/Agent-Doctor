package memory

type Item struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	Content    string `json:"content"`
	State      string `json:"state"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}
type CreateInput struct {
	Content    string `json:"content"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId,omitempty"`
}
type UpdateInput struct {
	Content string `json:"content,omitempty"`
	State   string `json:"state,omitempty"`
}
