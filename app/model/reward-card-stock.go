package model

import (
	"errors"
	"inis/app/facade"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// ============================== 奖励卡密库存（纯卡密） ==============================
//
// 场景：签到这类活动要发「卡密」，但卡密是管理员自己提供的一串码（外部渠道兑换码、
// 自己印制的码…），**与积分无关**，只是把码交给用户。因此单独一张表当「库存」用：
//
//   - 一行 = 一张卡密（Code 唯一）；
//   - Status：0 未发放 / 1 已发放（发放时绑定 uid 与时间）；
//   - 库存按「配置里当前填写的卡密」实时圈定：发放只从这些码里挑未发放的那张，
//     所以管理员把某个码从配置里删掉，它就不会再被发出去（已经发出去的不受影响）；
//   - 发完即失效：没有未发放的码时本次奖励什么都不发（不做任何降级 / 改发）。
//
// 与「积分卡密」（inis_integral_card）的区别：这里的卡密没有面额、不参与积分兑换流程，
// 也不占用积分卡密池的统计。

// 奖励卡密状态
const (
	RewardCardStatusUnused  = 0 // 未发放（还在库存里）
	RewardCardStatusGranted = 1 // 已发放（已经发给某个用户）
)

// 奖励卡密长度限制（太短容易抄错，也容易被撞码）
const (
	RewardCardMinLength = 8
	RewardCardMaxLength = 64
	// RewardCardMaxCount 单个奖励项可配置的卡密数量上限
	RewardCardMaxCount = 1000
)

// RewardCard - 奖励卡密库存表
type RewardCard struct {
	Id int `gorm:"type:int(32); comment:主键;" json:"id"`
	// Code 卡密（唯一索引防重）
	Code string `gorm:"size:64; uniqueIndex:uk_reward_card; comment:卡密（唯一）;" json:"code"`
	// Status 状态（0 未发放 1 已发放）
	Status int `gorm:"tinyint; index; comment:状态（0未发放 1已发放）; default:0;" json:"status"`
	// Uid 领取该卡密的用户ID
	Uid int `gorm:"type:int(32); index; comment:领取用户ID; default:0;" json:"uid"`
	// GrantTime 发放时间（秒级时间戳）
	GrantTime int64 `gorm:"comment:发放时间; default:0;" json:"grant_time"`
	// Remark 备注（来源说明）
	Remark string `gorm:"size:255; comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitRewardCard - 初始化奖励卡密库存表
// 索引通过结构体 tag 声明（唯一索引、状态、领取人），由 AutoMigrate 创建；
// 不使用 "CREATE INDEX IF NOT EXISTS"，MySQL 不支持该语法（会静默失败）。
func InitRewardCard() {
	if err := facade.DB.Drive().AutoMigrate(&RewardCard{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "RewardCard表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *RewardCard) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// ErrRewardCardEmpty - 库存里没有未发放的卡密
//
// 这是「库存走完」的正常状态而不是故障：调用方（卡密奖励资产）捕获它之后
// 本次奖励什么都不发（不降级、不改发），签到本身仍然成功。
var ErrRewardCardEmpty = errors.New("卡密库存已发完！")

// normalizeRewardCard - 卡密标准化（去空格 / 连字符并转大写）
//
// 用户抄写卡密时常多带空格或连字符，这里与展示、后台填写口径统一，
// 避免「一模一样却兑不了」。
func normalizeRewardCard(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")
	return strings.ToUpper(code)
}

// ParseCardCodes - 把「管理员填写的卡密」解析成字符串列表
//
// 积分卡密的自定义导入与奖励卡密库存共用（两处都是「贴一段文本 / 传一个数组」）：
//   - 数组：[]string / []any
//   - 文本：一行一个，也支持英文逗号 / 中文逗号 / 分号分隔；空行与首尾空白忽略
func ParseCardCodes(raw any) []string {
	result := make([]string, 0)

	switch value := raw.(type) {
	case nil:
		return result
	case string:
		fields := strings.FieldsFunc(value, func(r rune) bool {
			switch r {
			case '\n', '\r', ',', '，', ';', '；':
				return true
			}
			return false
		})
		for _, item := range fields {
			if code := strings.TrimSpace(item); code != "" {
				result = append(result, code)
			}
		}
		return result
	}

	for _, item := range cast.ToStringSlice(raw) {
		if code := strings.TrimSpace(item); code != "" {
			result = append(result, code)
		}
	}

	return result
}

// cleanRewardCardCodes - 标准化 + 去重 + 长度校验，返回可入库的卡密列表
func cleanRewardCardCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	result := make([]string, 0, len(codes))

	for _, item := range codes {
		code := normalizeRewardCard(item)
		if code == "" {
			continue
		}
		if len(code) < RewardCardMinLength || len(code) > RewardCardMaxLength {
			facade.Log.Warn(map[string]any{"code": code}, "奖励卡密长度不合法，已跳过")
			continue
		}
		if _, exist := seen[code]; exist {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}

	return result
}

// EnsureRewardCards - 把配置里的卡密补进库存（可在事务内调用）
//
// 语义：
//   - 已存在的卡密一律不动（不覆盖状态，也不覆盖备注）；
//   - 只插入缺失的那些（首次发放、或管理员新加了卡密时）；
//   - 并发下（唯一索引冲突）不当作错误 —— 另一处已经插进去了。
//
// 返回本次新入库的数量。
func EnsureRewardCards(tx *gorm.DB, codes []string, remark string) (inserted int, err error) {
	if len(codes) == 0 {
		return 0, nil
	}
	if tx == nil {
		tx = facade.DB.Drive()
	}

	clean := cleanRewardCardCodes(codes)
	if len(clean) == 0 {
		return 0, nil
	}

	// 已存在的卡密（用 Raw 查，绕过软删除过滤：回收站里的卡密同样占用卡号）
	existing := make(map[string]struct{}, len(clean))
	for start := 0; start < len(clean); start += 500 {
		end := start + 500
		if end > len(clean) {
			end = len(clean)
		}

		var rows []string
		if err = tx.Raw("SELECT code FROM inis_reward_card WHERE code IN ?", clean[start:end]).
			Scan(&rows).Error; err != nil {
			facade.Log.Error(map[string]any{"error": err.Error()}, "查询奖励卡密库存失败")
			return 0, err
		}
		for _, item := range rows {
			existing[item] = struct{}{}
		}
	}

	pending := make([]RewardCard, 0, len(clean))
	for _, code := range clean {
		if _, exist := existing[code]; exist {
			continue
		}
		pending = append(pending, RewardCard{
			Code:   code,
			Status: RewardCardStatusUnused,
			Remark: remark,
		})
	}

	if len(pending) == 0 {
		return 0, nil
	}

	if err = tx.Create(&pending).Error; err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return 0, nil
		}
		facade.Log.Error(map[string]any{"error": err.Error()}, "奖励卡密入库失败")
		return 0, err
	}

	return len(pending), nil
}

// GrantRewardCardTx - 从库存里发一张卡密给用户（在事务内执行）
//
// 只从 codes 指定的那些卡密里挑「未发放」的一张（codes 为空视为没有库存）：
//   - 先补齐库存（配置里新增的卡密在首次发放时入库）；
//   - 再用「状态条件更新 + 影响行数」原子占用，并发下同一张不会发两次；
//   - 全部发完时返回 ErrRewardCardEmpty，由调用方决定「什么都不发」。
//
// 返回：卡密明文与发放信息（会回填到奖励明细的 extra 里）。
func GrantRewardCardTx(tx *gorm.DB, uid int, codes []string) (facade.H, error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}
	if tx == nil {
		tx = facade.DB.Drive()
	}

	clean := cleanRewardCardCodes(codes)
	if len(clean) == 0 {
		return nil, ErrRewardCardEmpty
	}

	// 配置里新增的卡密先入库（幂等）
	if _, err := EnsureRewardCards(tx, clean, "奖励卡密"); err != nil {
		return nil, err
	}

	// 被并发抢走时换下一张重试
	for attempt := 0; attempt < 3; attempt++ {

		var record RewardCard

		query := tx.Where("code IN ?", clean).Where("status = ?", RewardCardStatusUnused)
		if err := query.Order("id asc").First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrRewardCardEmpty
			}
			return nil, err
		}

		now := time.Now().Unix()
		occupied := tx.Model(&RewardCard{}).
			Where("id = ? AND status = ?", record.Id, RewardCardStatusUnused).
			Updates(map[string]any{
				"status":     RewardCardStatusGranted,
				"uid":        uid,
				"grant_time": now,
			})
		if occupied.Error != nil {
			return nil, occupied.Error
		}
		if occupied.RowsAffected == 0 {
			continue
		}

		return facade.H{
			"card_id":    record.Id,
			"card":       record.Code,
			"granted_at": now,
		}, nil
	}

	return nil, ErrRewardCardEmpty
}

// RewardCardStock - 库存统计（给定配置里填写的卡密 → 总数 / 已发 / 剩余）
//
// 用于后台「签到设置」页显示每个卡密奖励还剩多少张：会先把缺失的卡密补进库存，
// 这样 total 就等于管理员填写的数量（不用自己数），remain 就是还能发几张。
func RewardCardStock(codes []string) facade.H {
	clean := cleanRewardCardCodes(codes)

	result := facade.H{
		"total":  len(clean),
		"issued": 0,
		"remain": 0,
	}

	if len(clean) == 0 {
		return result
	}

	if _, err := EnsureRewardCards(nil, clean, "奖励卡密"); err != nil {
		return result
	}

	var rows []map[string]any
	facade.DB.Drive().Raw(
		"SELECT status, COUNT(*) AS count FROM inis_reward_card "+
			"WHERE code IN ? AND (delete_time IS NULL OR delete_time = 0) GROUP BY status",
		clean,
	).Scan(&rows)

	for _, row := range rows {
		count := cast.ToInt(row["count"])
		if cast.ToInt(row["status"]) == RewardCardStatusGranted {
			result["issued"] = count
			continue
		}
		result["remain"] = count
	}

	return result
}
