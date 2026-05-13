package projex

// WebhookPayload 是 Projex 自动化 Webhook 的最小解析模型。
// 云效 Projex 自动化规则 Webhook 的 Body 通常是 GetWorkitem 接口语义下的工作项数据。
type WebhookPayload struct {
	CategoryID        string             `json:"categoryId"`
	ID                string             `json:"id"`
	IDPath            string             `json:"idPath"`
	Identifier        string             `json:"identifier"`
	SerialNumber      string             `json:"serialNumber"`
	Subject           string             `json:"subject"`
	Description       string             `json:"description"`
	FormatType        string             `json:"formatType"`
	LogicalStatus     string             `json:"logicalStatus"`
	Status            NamedValue         `json:"status"`
	AssignedTo        NamedValue         `json:"assignedTo"`
	Creator           NamedValue         `json:"creator"`
	Modifier          NamedValue         `json:"modifier"`
	Verifier          NamedValue         `json:"verifier"`
	Space             NamedValue         `json:"space"`
	Sprint            NamedValue         `json:"sprint"`
	WorkitemType      NamedValue         `json:"workitemType"`
	ParentID          string             `json:"parentId"`
	Parent            NamedValue         `json:"parent"`
	Participants      []NamedValue       `json:"participants"`
	Trackers          []NamedValue       `json:"trackers"`
	Versions          []NamedValue       `json:"versions"`
	Labels            []NamedValue       `json:"labels"`
	CustomFieldValues []CustomFieldValue `json:"customFieldValues"`
	GmtCreate         string             `json:"gmtCreate"`
	GmtModified       string             `json:"gmtModified"`
	UpdateStatusAt    string             `json:"updateStatusAt"`
}

// NamedValue 是 Projex payload 中常见的 ID + 名称对象。
type NamedValue struct {
	ID            string `json:"id"`
	Identifier    string `json:"identifier"`
	Name          string `json:"name"`
	NameEn        string `json:"nameEn"`
	DisplayName   string `json:"displayName"`
	Color         string `json:"color"`
	StatusStageID string `json:"statusStageId"`
}

// CustomFieldValue 表示 Projex 工作项自定义字段。
type CustomFieldValue struct {
	FieldID         string             `json:"fieldId"`
	FieldIdentifier string             `json:"fieldIdentifier"`
	FieldName       string             `json:"fieldName"`
	FieldFormat     string             `json:"fieldFormat"`
	Value           any                `json:"value"`
	Values          []CustomFieldEntry `json:"values"`
}

// CustomFieldEntry 表示 Projex 自定义字段 values 数组中的单个值。
type CustomFieldEntry struct {
	DisplayValue string `json:"displayValue"`
	Identifier   string `json:"identifier"`
}
