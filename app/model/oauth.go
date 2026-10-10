package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"

	"inis/app/facade"
)

// ============================== 第三方登录（QQ / GitHub / Gitee） ==============================
//
// 两块数据：
//  1. 应用配置：存在 config 表的一条记录里（key = SYSTEM_OAUTH，json 存各平台的 app_id/app_key/redirect），
//     由后台「系统配置 → 第三方登录」维护，路径 /api/config/save；
//  2. 绑定关系：独立表 inis_user_oauth（见 UserOauth），一个第三方账号只能绑一个本站用户，
//     一个本站用户同一平台也只能绑一个第三方账号。
//
// 平台差异：
//   - QQ    ：OAuth2.0 + openid/unionid 两段式，换取用户信息需要 app_id + openid，必须带 redirect_uri；
//   - GitHub：标准 OAuth2.0，openid 用数字 id，必须带 User-Agent 请求头；
//   - Gitee ：标准 OAuth2.0，换取 token 需要 redirect_uri。

// OauthConfigKey - 第三方登录配置在 config 表的键
const OauthConfigKey = "SYSTEM_OAUTH"

// 平台标识（与前端 method 名、auth_rules 路由保持一致）
const (
	OauthPlatformQQ     = "qq"
	OauthPlatformGithub = "github"
	OauthPlatformGitee  = "gitee"
)

// oauthPlatformList - 支持的平台（顺序与后台展示、前端按钮一致）
var oauthPlatformList = []string{OauthPlatformQQ, OauthPlatformGithub, OauthPlatformGitee}

// oauthPlatformNames - 平台展示名
var oauthPlatformNames = map[string]string{
	OauthPlatformQQ:     "QQ",
	OauthPlatformGithub: "GitHub",
	OauthPlatformGitee:  "Gitee",
}

// oauthPlatformAuthorize - 授权页地址 + 需要的 scope（前端据此拼接跳转链接，避免把平台细节散落到前端）
var oauthPlatformAuthorize = map[string]map[string]string{
	OauthPlatformQQ: {
		"url":   "https://graph.qq.com/oauth2.0/authorize",
		"scope": "get_user_info",
	},
	OauthPlatformGithub: {
		"url":   "https://github.com/login/oauth/authorize",
		"scope": "read:user",
	},
	OauthPlatformGitee: {
		"url":   "https://gitee.com/oauth/authorize",
		"scope": "user_info",
	},
}

// oauthPlatformRedirectRequired - 换取 token 时是否必须带上 redirect_uri（必须与授权时一致）
// GitHub 允许省略，QQ / Gitee 必须带，否则换 token 会失败。
var oauthPlatformRedirectRequired = map[string]bool{
	OauthPlatformQQ:     true,
	OauthPlatformGithub: false,
	OauthPlatformGitee:  true,
}

// OauthPlatformList - 支持的平台列表（对外暴露，控制器与后台都用它遍历）
func OauthPlatformList() []string {
	return oauthPlatformList
}

// OauthPlatformName - 平台展示名
func OauthPlatformName(platform string) string {
	if name, ok := oauthPlatformNames[platform]; ok {
		return name
	}
	return platform
}

// OauthPlatformValid - 是否为受支持的平台
func OauthPlatformValid(platform string) bool {
	_, ok := oauthPlatformNames[platform]
	return ok
}

// OauthPlatformAuthorize - 授权页地址与 scope
func OauthPlatformAuthorize(platform string) map[string]string {
	if item, ok := oauthPlatformAuthorize[platform]; ok {
		return item
	}
	return map[string]string{"url": "", "scope": ""}
}

// OauthRedirectRequired - 该平台换取 token 是否需要 redirect_uri
func OauthRedirectRequired(platform string) bool {
	return oauthPlatformRedirectRequired[platform]
}

// defaultOauthSettings - 默认配置
//
//   - 三个平台默认都是「关闭」：没在后台填 AppID / AppKey / 回调地址之前，
//     前端不应该出现点了也没用的登录按钮（OauthPlatformAvailable 会一并判定）；
//   - QQ / GitHub 的默认值来自旧实现（app/api/controller/oauth.go 里硬编码的应用），
//     这里作为默认值保留，站长在后台改成自己的应用即可；
//   - auto_register：是否允许用第三方账号创建新账号。
//     开启：未绑定时回调页会给出「创建新账号」选项（仍需用户确认，不会静默建号）；
//     关闭：只提供「绑定已有账号」（输入账号密码），未注册的用户请走常规注册。
func defaultOauthSettings() facade.H {
	return facade.H{
		OauthPlatformQQ: facade.H{
			"enable":   0,
			"app_id":   "102045704",
			"app_key":  "28BAGLMLGHUdijgY",
			"redirect": "",
		},
		OauthPlatformGithub: facade.H{
			"enable":   0,
			"app_id":   "Iv1.cc978ca4f5d98345",
			"app_key":  "cca6c49fdeb724f341bae59ccc1e400de12e6c63",
			"redirect": "",
		},
		OauthPlatformGitee: facade.H{
			"enable":   0,
			"app_id":   "",
			"app_key":  "",
			"redirect": "",
		},
		"auto_register": 1,
		// timeout：第三方接口请求超时（秒）。自建服务器出网慢时，经常是它先触发
		"timeout": 15,
		// proxy：第三方接口请求走哪个代理（http / https / socks5），留空表示直连。
		// 例：http://127.0.0.1:7890 —— GitHub 在部分网络环境下直连不通，需要走代理
		"proxy": "",
	}
}

// OauthSettings - 读取第三方登录配置（缓存优先，缺失项用默认值兜底）
//
// 缓存名与 config/save 清理的名字保持一致（config[KEY]），否则后台改完要等重启才生效。
func OauthSettings() facade.H {
	settings := defaultOauthSettings()

	cacheName := "config[" + OauthConfigKey + "]"

	if facade.Cache.Has(cacheName) {
		if data, ok := facade.Cache.Get(cacheName).(map[string]any); ok {
			return mergeOauthSettings(settings, data)
		}
		if data, ok := facade.Cache.Get(cacheName).(facade.H); ok {
			return mergeOauthSettings(settings, data)
		}
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", OauthConfigKey).Find()
	if !utils.Is.Empty(item) {
		if data, ok := item["json"].(map[string]any); ok {
			settings = mergeOauthSettings(settings, data)
			facade.Cache.Set(cacheName, settings)
		}
	}

	return settings
}

// OauthPlatformConfig - 指定平台的配置（已与默认值合并）
func OauthPlatformConfig(platform string) facade.H {
	settings := OauthSettings()
	if item, ok := settings[platform].(map[string]any); ok {
		return facade.H(item)
	}
	if item, ok := settings[platform].(facade.H); ok {
		return item
	}
	return facade.H{}
}

// OauthAutoRegister - 是否允许用第三方账号创建新账号（SYSTEM_OAUTH.auto_register）
func OauthAutoRegister() bool {
	return cast.ToInt(OauthSettings()["auto_register"]) == 1
}

// OauthTimeout - 第三方接口请求超时（秒 → Duration），限制在 3~60 秒
func OauthTimeout() time.Duration {
	return time.Duration(oauthTimeoutValue(OauthSettings()["timeout"])) * time.Second
}

// OauthProxy - 第三方接口请求使用的代理地址（空表示直连）
func OauthProxy() string {
	return strings.TrimSpace(cast.ToString(OauthSettings()["proxy"]))
}

// oauthTimeoutValue - 超时值归一化（3~60 秒，缺省 15 秒）
func oauthTimeoutValue(value any) int {
	seconds := cast.ToInt(value)
	if seconds <= 0 {
		return 15
	}
	if seconds < 3 {
		return 3
	}
	if seconds > 60 {
		return 60
	}
	return seconds
}

// OauthPlatformAvailable - 平台是否可用（已开启 + 必填项齐全）
//
// 判定不通过时返回原因，控制器直接把原因回给前端 —— 免得出现「按钮点下去只报一句获取失败」。
func OauthPlatformAvailable(platform string) (bool, string) {
	if !OauthPlatformValid(platform) {
		return false, "不支持的第三方登录方式！"
	}

	config := OauthPlatformConfig(platform)

	if cast.ToInt(config["enable"]) != 1 {
		return false, fmt.Sprintf("%v 登录未开启！", OauthPlatformName(platform))
	}
	if utils.Is.Empty(config["app_id"]) {
		return false, fmt.Sprintf("后台未配置 %v 的 AppID！", OauthPlatformName(platform))
	}
	if utils.Is.Empty(config["app_key"]) {
		return false, fmt.Sprintf("后台未配置 %v 的 AppKey！", OauthPlatformName(platform))
	}
	if OauthRedirectRequired(platform) && utils.Is.Empty(config["redirect"]) {
		return false, fmt.Sprintf("后台未配置 %v 的回调地址！", OauthPlatformName(platform))
	}

	return true, ""
}

// mergeOauthSettings - 用配置值覆盖默认值（只认已知字段，避免脏数据影响判断）
func mergeOauthSettings(base facade.H, override map[string]any) facade.H {
	// 平台配置：enable 归一化为 0/1，其余字段原样取字符串
	for _, platform := range oauthPlatformList {
		raw, ok := override[platform]
		if !ok {
			continue
		}
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		current, _ := base[platform].(facade.H)
		if current == nil {
			current = facade.H{}
		}

		if value, ok := item["enable"]; ok {
			current["enable"] = oauthSwitchValue(value)
		}
		for _, key := range []string{"app_id", "app_key", "redirect"} {
			if value, ok := item[key]; ok {
				current[key] = cast.ToString(value)
			}
		}

		base[platform] = current
	}

	if value, ok := override["timeout"]; ok {
		base["timeout"] = oauthTimeoutValue(value)
	}

	if value, ok := override["proxy"]; ok {
		base["proxy"] = cast.ToString(value)
	}

	if value, ok := override["auto_register"]; ok {
		base["auto_register"] = oauthSwitchValue(value)
	}

	return base
}

// oauthSwitchValue - 开关值归一化为 0/1
//
// 兼容三种来源：后台下拉的 "0"/"1" 字符串、数字、以及手写 JSON 时的布尔值。
func oauthSwitchValue(value any) int {
	if flag, ok := value.(bool); ok {
		if flag {
			return 1
		}
		return 0
	}
	if cast.ToInt(value) == 1 {
		return 1
	}
	return 0
}

// ============================== 绑定关系 ==============================

// UserOauth - 第三方账号绑定表
//
// 字段取舍：只存「身份 + 展示用的昵称头像」，不存 access_token / refresh_token ——
// 登录只需要 openid，长期保存第三方令牌等于多留一份泄露面（将来要拉资料再另说）。
type UserOauth struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); index:idx_oauth_uid; comment:本站用户ID;" json:"uid"`
	Platform string `gorm:"size:32; uniqueIndex:idx_oauth_platform_openid; comment:平台标识（qq/github/gitee）;" json:"platform"`
	Openid   string `gorm:"size:128; uniqueIndex:idx_oauth_platform_openid; comment:平台用户唯一标识（GitHub 为数字 id）;" json:"openid"`
	Unionid  string `gorm:"size:128; comment:开放平台 UnionID（QQ 多应用打通，其它平台为空）;" json:"unionid"`
	Nickname string `gorm:"size:128; comment:第三方昵称（仅展示，改昵称去个人资料）;" json:"nickname"`
	Avatar   string `gorm:"comment:第三方头像（仅展示）;" json:"avatar"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitUserOauth - 初始化UserOauth表
func InitUserOauth() {
	err := facade.DB.Drive().AutoMigrate(&UserOauth{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserOauth表迁移失败")
		return
	}
	// 兼容旧库：AutoMigrate 在已有表上可能不自动创建 uniqueIndex
	// 一个第三方账号只能绑定一个本站用户（解绑是软删除，重新绑定时会复用该行，见 BindUserOauth）
	facade.DB.Drive().Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_platform_openid ON inis_user_oauth(platform, openid)")
}

// AfterFind - 查询后的钩子
func (this *UserOauth) AfterFind(tx *gorm.DB) (err error) {
	this.Json = utils.Json.Decode(this.Json)
	return
}

// FindUserOauth - 按「平台 + openid」查绑定关系（含回收站，解绑后重新绑定要复用同一行）
func FindUserOauth(platform string, openid string) facade.H {
	if utils.Is.Empty(platform) || utils.Is.Empty(openid) {
		return nil
	}
	item, _ := facade.DB.Model(&UserOauth{}).WithTrashed().
		Where("platform", platform).
		Where("openid", openid).
		Find()
	return item
}

// FindUserOauthByUid - 查某个用户在指定平台的绑定关系（只取生效中的）
func FindUserOauthByUid(uid int, platform string) facade.H {
	if uid <= 0 || utils.Is.Empty(platform) {
		return nil
	}
	item, _ := facade.DB.Model(&UserOauth{}).
		Where("uid", uid).
		Where("platform", platform).
		Find()
	return item
}

// UserOauthList - 某个用户的全部绑定（只取生效中的，供「账号安全」页展示）
func UserOauthList(uid int) []map[string]any {
	if uid <= 0 {
		return []map[string]any{}
	}
	// 注意：facade 的 Select() 要求 dest 是切片（传单结构体会恒返回空）
	var table []UserOauth
	list, _ := facade.DB.Model(&table).Where("uid", uid).Order("id asc").Select()
	return list
}

// UserOauthMap - 批量查询「一批用户」的第三方绑定（用户列表用，避免 N+1）
//
// 返回：uid -> [{platform, name, nickname, avatar, create_time}]
// 说明：调用方需自行确认有权限（后台用户列表只在管理员视角附加该数据）
func UserOauthMap(uids []int) map[int][]facade.H {

	result := map[int][]facade.H{}
	if len(uids) == 0 {
		return result
	}

	// 注意：facade 的 Select() 要求 dest 是切片（传单结构体会恒返回空）
	var table []UserOauth
	list, _ := facade.DB.Model(&table).Where("uid", "in", uids).Order("id asc").Select()

	for _, item := range list {
		uid := cast.ToInt(item["uid"])
		platform := cast.ToString(item["platform"])
		result[uid] = append(result[uid], facade.H{
			"platform":    platform,
			"name":        OauthPlatformName(platform),
			"nickname":    item["nickname"],
			"avatar":      item["avatar"],
			"create_time": item["create_time"],
		})
	}

	return result
}
// UserOauthBoundCount - 某个用户已绑定的平台数量（判断解绑后是否还有别的登录方式）
func UserOauthBoundCount(uid int) int {
	return len(UserOauthList(uid))
}

// BindUserOauth - 绑定第三方账号
//
// 三种情况一次处理干净：
//  1. 该第三方账号已生效：属于自己 → 刷新昵称头像；属于别人 → 报错（一个第三方账号只能绑一个本站用户）；
//  2. 该第三方账号存在于回收站（曾解绑）：复用该行（uniqueIndex 不允许重建，且能保留创建时间）；
//  3. 全新绑定：当前用户在**同一平台**已有别的绑定 → 报错，让用户先解绑（避免一个平台多号互相顶替）。
func BindUserOauth(uid int, platform, openid, unionid, nickname, avatar string) (facade.H, error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}
	if !OauthPlatformValid(platform) {
		return nil, errors.New("不支持的第三方登录方式！")
	}
	if utils.Is.Empty(openid) {
		return nil, errors.New("第三方账号标识为空，绑定失败！")
	}

	now := time.Now().Unix()
	update := map[string]any{
		"uid":         uid,
		"unionid":     unionid,
		"nickname":    nickname,
		"avatar":      avatar,
		"delete_time": 0,
	}

	exist := FindUserOauth(platform, openid)
	if !utils.Is.Empty(exist) {
		existUid := cast.ToInt(exist["uid"])
		deleted := cast.ToInt(exist["delete_time"]) > 0

		// 已被别人绑定（且未解绑）
		if !deleted && existUid != uid {
			return nil, fmt.Errorf("该%v账号已绑定其它用户，请换一个账号或先在原账号中解绑！", OauthPlatformName(platform))
		}

		// 自己重复绑定 / 解绑后重新绑定：复用同一行
		if _, err := facade.DB.Model(&UserOauth{}).WithTrashed().Where("id", exist["id"]).Update(update); err != nil {
			return nil, errors.New("绑定失败，请稍后重试！")
		}
		return FindUserOauth(platform, openid), nil
	}

	// 同一平台只能绑一个第三方账号
	if bound := FindUserOauthByUid(uid, platform); !utils.Is.Empty(bound) {
		return nil, fmt.Errorf("当前账号已绑定其它%v账号（%v），请先解绑！", OauthPlatformName(platform), cast.ToString(bound["nickname"]))
	}

	table := UserOauth{
		Uid:        uid,
		Platform:   platform,
		Openid:     openid,
		Unionid:    unionid,
		Nickname:   nickname,
		Avatar:     avatar,
		CreateTime: now,
		UpdateTime: now,
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		return nil, errors.New("绑定失败，请稍后重试！")
	}

	return FindUserOauth(platform, openid), nil
}

// UnbindUserOauth - 解绑第三方账号（软删除，重新绑定时复用该行）
func UnbindUserOauth(uid int, platform string) error {
	if uid <= 0 {
		return errors.New("请先登录！")
	}

	bound := FindUserOauthByUid(uid, platform)
	if utils.Is.Empty(bound) {
		return fmt.Errorf("当前账号未绑定%v！", OauthPlatformName(platform))
	}

	if _, err := facade.DB.Model(&UserOauth{}).Where("id", bound["id"]).Delete(); err != nil {
		return errors.New("解绑失败，请稍后重试！")
	}

	return nil
}
