package validator

type Attachment struct {
	TargetType string `json:"target_type" rule:"required"`
	TargetId   uint   `json:"target_id" rule:"number,min:0"`
	IsPublic   bool   `json:"is_public" rule:"bool"`
	Status     int8   `json:"status" rule:"number,min:0,max:1"`
}

var AttachmentMessage = map[string]string{
	"target_type.required": "关联业务类型不能为空！",
	"target_id.number":     "关联业务ID必须是数字！",
	"target_id.min":        "关联业务ID不能为负数！",
	"is_public.bool":       "是否公开必须是布尔值！",
	"status.number":        "状态必须是数字！",
	"status.min":           "状态值无效！",
	"status.max":           "状态值无效！",
}

func (this Attachment) Message() map[string]string {
	return AttachmentMessage
}

func (this Attachment) Struct() any {
	return this
}