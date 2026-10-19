package validator

// Docs - 文档页面
type Docs struct {
	Title   string `json:"title" rule:"required"`
	Slug    string `json:"slug" rule:"required,alphaDash"`
	Version string `json:"version" rule:"required"`
	Sort    int    `json:"sort" rule:"number,min:0"`
	Status  int    `json:"status" rule:"number,min:0,max:1"`
}

var DocsMessage = map[string]string{
	"title.required":  "文档标题不能为空",
	"slug.required":   "页面标识（slug）不能为空",
	"slug.alphaDash":  "页面标识只能由字母、数字、破折号（ - ）以及下划线（ _ ）组成",
	"version.required": "版本不能为空",
	"sort.number":     "排序只能是数字",
	"sort.min":        "排序不能小于 0",
	"status.number":   "状态只能是数字",
	"status.min":      "状态只能是 0 或 1",
	"status.max":      "状态只能是 0 或 1",
}

func (this Docs) Message() map[string]string {
	return DocsMessage
}

func (this Docs) Struct() any {
	return this
}

// DocsGroup - 文档目录
type DocsGroup struct {
	Name    string `json:"name" rule:"required"`
	Slug    string `json:"slug" rule:"required,alphaDash"`
	Version string `json:"version" rule:"required"`
	Sort    int    `json:"sort" rule:"number,min:0"`
	Status  int    `json:"status" rule:"number,min:0,max:1"`
}

var DocsGroupMessage = map[string]string{
	"name.required":     "目录名称不能为空",
	"slug.required":     "目录标识（slug）不能为空",
	"slug.alphaDash":    "目录标识只能由字母、数字、破折号（ - ）以及下划线（ _ ）组成",
	"version.required":  "版本不能为空",
	"sort.number":       "排序只能是数字",
	"sort.min":          "排序不能小于 0",
	"status.number":     "状态只能是数字",
	"status.min":        "状态只能是 0 或 1",
	"status.max":        "状态只能是 0 或 1",
}

func (this DocsGroup) Message() map[string]string {
	return DocsGroupMessage
}

func (this DocsGroup) Struct() any {
	return this
}

// DocsProject - 文档项目
type DocsProject struct {
	Name string `json:"name" rule:"required"`
	Slug string `json:"slug" rule:"required,alphaDash"`
	Sort int    `json:"sort" rule:"number,min:0"`
	// 状态：1-启用 0-禁用
	Status int `json:"status" rule:"number,min:0,max:1"`
	// 可见性：1-公开 2-登录可见 3-指定角色组
	Visible int `json:"visible" rule:"number,min:1,max:3"`
	// 启用评论：0-禁止 1-允许
	CommentEnabled int `json:"comment_enabled" rule:"number,min:0,max:1"`
}

var DocsProjectMessage = map[string]string{
	"name.required":           "项目名称不能为空",
	"slug.required":           "项目别名（slug）不能为空",
	"slug.alphaDash":          "项目别名只能由字母、数字、破折号（ - ）以及下划线（ _ ）组成",
	"sort.number":             "排序只能是数字",
	"sort.min":                "排序不能小于 0",
	"status.number":           "状态只能是数字",
	"status.min":              "状态只能是 0 或 1",
	"status.max":              "状态只能是 0 或 1",
	"visible.number":          "可见性只能是数字",
	"visible.min":             "可见性只能是 1（公开）、2（登录可见）或 3（指定角色组）",
	"visible.max":             "可见性只能是 1（公开）、2（登录可见）或 3（指定角色组）",
	"comment_enabled.number":  "启用评论只能是数字",
	"comment_enabled.min":     "启用评论只能是 0（禁止）或 1（允许）",
	"comment_enabled.max":     "启用评论只能是 0（禁止）或 1（允许）",
}

func (this DocsProject) Message() map[string]string {
	return DocsProjectMessage
}

func (this DocsProject) Struct() any {
	return this
}
