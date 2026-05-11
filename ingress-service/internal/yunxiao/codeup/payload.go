package codeup

// WebhookPayload 是 Codeup Webhook 的最小解析模型。
// 当前只提取标准化事件所需字段，完整原始 payload 仍保存在 raw_event。
type WebhookPayload struct {
	ObjectKind       string                  `json:"object_kind"`
	EventName        string                  `json:"event_name"`
	ProjectID        int64                   `json:"project_id"`
	User             User                    `json:"user"`
	Repository       Repository              `json:"repository"`
	ObjectAttributes MergeRequestAttributes  `json:"object_attributes"`
	MergeRequest     *MergeRequestAttributes `json:"merge_request"`
	Commit           *Commit                 `json:"commit"`
}

// User 是 Codeup Webhook 中的触发用户。
type User struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Repository 是 Codeup Webhook 中的代码库信息。
type Repository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	GitHTTPURL  string `json:"git_http_url"`
	GitSSHURL   string `json:"git_ssh_url"`
	Homepage    string `json:"homepage"`
	Description string `json:"description"`
}

// MergeRequestAttributes 是 Codeup 合并请求事件的核心字段。
type MergeRequestAttributes struct {
	ID              int64   `json:"id"`
	LocalID         int64   `json:"local_id"`
	BizID           string  `json:"biz_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Action          string  `json:"action"`
	State           string  `json:"state"`
	SourceBranch    string  `json:"source_branch"`
	TargetBranch    string  `json:"target_branch"`
	SourceProjectID int64   `json:"source_project_id"`
	TargetProjectID int64   `json:"target_project_id"`
	LastCommit      *Commit `json:"last_commit"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// Commit 是 Codeup Webhook 中的 commit 信息。
type Commit struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
}
