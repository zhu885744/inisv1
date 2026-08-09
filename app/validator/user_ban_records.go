package validator

type UserBanRecords struct {
	Uid           int    `json:"uid" rule:"number"`
	BanType       int    `json:"ban_type" rule:"number,min:0,max:31"`
	Reason        string `json:"reason" rule:"max:512"`
	Duration      int    `json:"duration" rule:"number,min:0"`
	DeleteContent int    `json:"delete_content" rule:"number,min:0,max:1"`
	BanAppeal     int    `json:"ban_appeal" rule:"number,min:0,max:1"`
	FreezeUser    int    `json:"freeze_user" rule:"number,min:0,max:1"`
}

var UserBanRecordsMessage = map[string]string{
	"uid.number":            "uid 必须为数字！",
	"ban_type.number":       "封禁类型必须为数字！",
	"ban_type.min":          "封禁类型不能小于0！",
	"ban_type.max":          "封禁类型不能大于31！",
	"reason.max":            "封禁原因不能超过512个字符！",
	"duration.number":       "封禁时长必须为数字！",
	"duration.min":          "封禁时长不能小于0！",
	"delete_content.number": "删除内容参数必须为数字！",
	"delete_content.min":    "删除内容参数不能小于0！",
	"delete_content.max":    "删除内容参数不能大于1！",
	"ban_appeal.number":     "禁止申诉参数必须为数字！",
	"ban_appeal.min":        "禁止申诉参数不能小于0！",
	"ban_appeal.max":        "禁止申诉参数不能大于1！",
	"freeze_user.number":    "冻结用户参数必须为数字！",
	"freeze_user.min":       "冻结用户参数不能小于0！",
	"freeze_user.max":       "冻结用户参数不能大于1！",
}

func (this UserBanRecords) Message() map[string]string {
	return UserBanRecordsMessage
}

func (this UserBanRecords) Struct() any {
	return this
}
