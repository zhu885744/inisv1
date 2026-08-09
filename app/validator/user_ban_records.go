package validator

type UserBanRecords struct {
	Uid      int    `json:"uid" rule:"number"`
	BanType  int    `json:"ban_type" rule:"number,min:0,max:31"`
	Reason   string `json:"reason" rule:"max:512"`
	Duration int    `json:"duration" rule:"number,min:0"`
}

var UserBanRecordsMessage = map[string]string{
	"uid.number":       "uid 必须为数字！",
	"ban_type.number":  "封禁类型必须为数字！",
	"ban_type.min":     "封禁类型不能小于0！",
	"ban_type.max":     "封禁类型不能大于31！",
	"reason.max":       "封禁原因不能超过512个字符！",
	"duration.number":  "封禁时长必须为数字！",
	"duration.min":     "封禁时长不能小于0！",
}

func (this UserBanRecords) Message() map[string]string {
	return UserBanRecordsMessage
}

func (this UserBanRecords) Struct() any {
	return this
}
