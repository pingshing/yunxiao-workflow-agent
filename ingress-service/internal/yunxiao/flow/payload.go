package flow

// WebhookPayload 是 Flow Webhook 通知插件的最小解析模型。
type WebhookPayload struct {
	Event        string        `json:"event"`
	Action       string        `json:"action"`
	Task         Task          `json:"task"`
	Sources      []Source      `json:"sources"`
	GlobalParams []GlobalParam `json:"globalParams"`
}

// Task 是 Flow Webhook 中的流水线任务信息。
type Task struct {
	PipelineID   string `json:"pipelineId"`
	PipelineName string `json:"pipelineName"`
	StageName    string `json:"stageName"`
	TaskName     string `json:"taskName"`
	BuildNumber  string `json:"buildNumber"`
	StatusCode   string `json:"statusCode"`
	PipelineURL  string `json:"pipelineUrl"`
	Message      string `json:"message"`
}

// Source 是 Flow Webhook 中的代码源信息。
type Source struct {
	Repo             string `json:"repo"`
	Branch           string `json:"branch"`
	CommitID         string `json:"commitId"`
	PreviousCommitID string `json:"privousCommitId"`
}

// GlobalParam 是 Flow Webhook 中的全局参数。
type GlobalParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
