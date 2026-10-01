package validator

// Users - 用户表字段校验
//
// 覆盖率要求：控制器 create / update 的 allow 白名单里出现过的字段，
// 必须在这里有对应规则，否则就是「接口允许写入、但完全不校验」的裸字段。
// 历史问题：phone / avatar / title / gender / json / text / remark / source
// 都允许写入却没有任何规则，可写入任意长度、任意格式的内容。
//
// 两个 go-utils 校验器的坑（已实测）：
//  1. 只识别 "max=32" 这种「等号」写法；"max:256"（冒号）既不是长度规则、
//     也不是内置规则，会被静默忽略 —— 等于没写。
//  2. 规则只在字段「存在且非空」时生效（required 除外），所以给可选字段
//     补规则是安全的，不会导致不传该字段的请求报错。
type Users struct {
	Account     string `json:"account" rule:"alphaNum,min=4,max=32"`
	Password    string `json:"password" rule:"min=6,max=32"`
	Email       string `json:"email" rule:"email,max=128"`
	Phone       string `json:"phone" rule:"mobile"`
	Nickname    string `json:"nickname" rule:"chsDash,max=32"`
	Description string `json:"description" rule:"max=256"`
	Title       string `json:"title" rule:"max=64"`
	Avatar      string `json:"avatar" rule:"max=512"`
	Gender      string `json:"gender" rule:"max=8"`
	Json        string `json:"json" rule:"max=65535"`
	Text        string `json:"text" rule:"max=65535"`
	Remark      string `json:"remark" rule:"max=256"`
	Source      string `json:"source" rule:"max=32"`
}

var UsersMessage = map[string]string{
	"account.alphaNum": "账号只能是字母和数字！",
	"account.min":      "账号最少不能少于4个字符！",
	"account.max":      "账号最多不能超过32个字符！",
	"password.min":     "密码最少不能少于6个字符！",
	"password.max":     "密码最多不能超过32个字符！",
	"email.email":      "邮箱格式不正确！",
	"email.max":        "邮箱最多不能超过128个字符！",
	"phone.mobile":     "手机号格式不正确！",
	"nickname.chsDash": "昵称只能是汉字、字母、数字和下划线_及破折号-！",
	"nickname.max":     "昵称最多不能超过32个字符！",
	"description.max":  "个人简介最多不能超过256个字符！",
	"title.max":        "头衔最多不能超过64个字符！",
	"avatar.max":       "头像地址最多不能超过512个字符！",
	"gender.max":       "性别取值不合法！",
	"json.max":         "配置数据过大，最多不能超过65535个字符！",
	"text.max":         "文本内容过大，最多不能超过65535个字符！",
	"remark.max":       "备注最多不能超过256个字符！",
	"source.max":       "注册来源最多不能超过32个字符！",
}

func (this Users) Message() map[string]string {
	return UsersMessage
}

func (this Users) Struct() any {
	return this
}
