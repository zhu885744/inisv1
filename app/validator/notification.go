package validator

type Notification struct {
	Type    string `json:"type" rule:"required"`
	Title   string `json:"title" rule:"required"`
	Content string `json:"content" rule:"required"`
}

var NotificationMessage = map[string]string{
	"type.required":    "通知类型不能为空！",
	"title.required":   "通知标题不能为空！",
	"content.required": "通知内容不能为空！",
}

func (this Notification) Message() map[string]string {
	return NotificationMessage
}

func (this Notification) Struct() any {
	return this
}
