package projex

// WebhookPayload 是 Projex 自动化 Webhook 的最小解析模型。
type WebhookPayload struct {
	ID                string             `json:"id"`
	Identifier        string             `json:"identifier"`
	Subject           string             `json:"subject"`
	Description       string             `json:"description"`
	Status            NamedValue         `json:"status"`
	AssignedTo        NamedValue         `json:"assignedTo"`
	Creator           NamedValue         `json:"creator"`
	Modifier          NamedValue         `json:"modifier"`
	Space             NamedValue         `json:"space"`
	Sprint            NamedValue         `json:"sprint"`
	WorkitemType      NamedValue         `json:"workitemType"`
	Labels            []NamedValue       `json:"labels"`
	CustomFieldValues []CustomFieldValue `json:"customFieldValues"`
	GmtCreate         string             `json:"gmtCreate"`
	GmtModified       string             `json:"gmtModified"`
}

// NamedValue 是 Projex payload 中常见的 ID + 名称对象。
type NamedValue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// CustomFieldValue 表示 Projex 工作项自定义字段。
type CustomFieldValue struct {
	FieldIdentifier string `json:"fieldIdentifier"`
	FieldName       string `json:"fieldName"`
	Value           any    `json:"value"`
}
