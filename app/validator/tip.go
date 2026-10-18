package validator

// Tip - 积分打赏
//
// bind_type 与 bind_id 描述「打赏的是什么」：article / moments / comment / page / user。
// 接收者由后端按内容作者解析，前端传的 to_uid 只用于校验、不参与决定收款人。
type Tip struct {
	BindType string `json:"bind_type" rule:"required"`
	BindId   int    `json:"bind_id" rule:"required,number"`
	Value    int    `json:"value" rule:"required,number,min:1"`
	// 字段名不能叫 Message：与 Valid 接口的 Message() 方法冲突，JSON 键仍为 message
	Remark string `json:"message"`
}

var TipMessage = map[string]string{
	"bind_type.required": "bind_type 不能为空！",
	"bind_id.required":   "bind_id 不能为空！",
	"bind_id.number":     "bind_id 必须为数字！",
	"value.required":     "打赏积分不能为空！",
	"value.number":       "打赏积分必须为数字！",
	"value.min":          "打赏积分不能少于 1 分！",
}

func (this Tip) Message() map[string]string {
	return TipMessage
}

func (this Tip) Struct() any {
	return this
}
