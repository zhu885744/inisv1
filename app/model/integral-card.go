package model

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"inis/app/facade"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 积分卡密常量
const (
	// IntegralCardStatusUnused 卡密未使用
	IntegralCardStatusUnused = 0
	// IntegralCardStatusUsed 卡密已使用
	IntegralCardStatusUsed = 1
	// IntegralCardMinLength 卡密最小长度（安全要求：不小于 8 位）
	IntegralCardMinLength = 8
	// IntegralCardMaxLength 卡密最大长度
	IntegralCardMaxLength = 64
	// IntegralCardDefaultLength 卡密默认长度
	IntegralCardDefaultLength = 16
	// IntegralCardMaxCount 单次生成数量上限
	IntegralCardMaxCount = 1000
	// IntegralCardMaxValue 单张卡密积分面额上限
	IntegralCardMaxValue = 1000000
	// IntegralCardCharset 卡密字符集（剔除易混淆字符 0/1/I/O，共 32 个字符，便于人工核对）
	IntegralCardCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

// IntegralCard - 积分卡密表
// 管理员批量生成卡密，用户凭卡密兑换积分；卡密长度不小于 8 位，支持设置积分面额与有效期（0 表示永久有效）
type IntegralCard struct {
	Id int `gorm:"type:int(32); comment:主键;" json:"id"`
	// Card 卡密（唯一索引防重，字符集不含易混淆字符）
	Card string `gorm:"size:64; uniqueIndex:uk_integral_card; comment:卡密（唯一）;" json:"card"`
	// Value 积分面额（正数）
	Value int `gorm:"type:int(32); index; comment:积分面额; default:0;" json:"value"`
	// Status 状态（0 未使用 1 已使用）
	Status int `gorm:"tinyint; index; comment:状态（0未使用 1已使用）; default:0;" json:"status"`
	// Batch 批次号（同一次生成共用，便于管理与统计）
	Batch string `gorm:"size:32; index; comment:批次号;" json:"batch"`
	// ExpireTime 过期时间（秒级时间戳，0 表示永久有效）
	ExpireTime int64 `gorm:"index; comment:过期时间（0=永久有效）; default:0;" json:"expire_time"`
	// Uid 使用该卡密的用户ID
	Uid int `gorm:"type:int(32); index; comment:使用用户ID;" json:"uid"`
	// UseTime 使用时间（秒级时间戳）
	UseTime int64 `gorm:"comment:使用时间; default:0;" json:"use_time"`
	// Remark 备注
	Remark string `gorm:"size:255; comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitIntegralCard - 初始化积分卡密表
// 索引通过结构体 tag 声明（唯一索引、状态、面额、批次、过期时间、用户），由 AutoMigrate 创建；
// 不使用 "CREATE INDEX IF NOT EXISTS"，MySQL 不支持该语法（会静默失败）。
func InitIntegralCard() {
	if err := facade.DB.Drive().AutoMigrate(&IntegralCard{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "IntegralCard表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *IntegralCard) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	this.Result = this.result()
	return
}

// result - 卡密展示结果（补充是否过期、是否可用等派生字段，方便前端展示）
func (this *IntegralCard) result() facade.H {
	now := time.Now().Unix()
	expired := this.ExpireTime > 0 && now > this.ExpireTime

	// available：仅未使用且未过期才可兑换
	available := this.Status == IntegralCardStatusUnused && !expired

	return facade.H{
		"expired":   expired,
		"available": available,
	}
}

// normalizeIntegralCard - 卡密标准化（去空格/连字符并转大写，提升用户输入容错）
func normalizeIntegralCard(card string) string {
	card = strings.TrimSpace(card)
	card = strings.ReplaceAll(card, " ", "")
	card = strings.ReplaceAll(card, "-", "")
	return strings.ToUpper(card)
}

// randomIntegralCard - 使用密码学安全随机数生成卡密
// 字符集长度为 32（2 的幂），通过位与运算取模可避免取模偏差，保证每个字符等概率出现
func randomIntegralCard(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := crand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for index, val := range buf {
		out[index] = IntegralCardCharset[int(val)&(len(IntegralCardCharset)-1)]
	}
	return string(out), nil
}

// randomIntegralBatch - 生成批次号随机后缀
func randomIntegralBatch() string {
	str, err := randomIntegralCard(6)
	if err != nil {
		return cast.ToString(time.Now().UnixNano() % 1000000)
	}
	return str
}

// maskIntegralCard - 卡密脱敏（用于流水/日志，避免明文泄露）
func maskIntegralCard(card string) string {
	if len(card) <= 4 {
		return "****"
	}
	return card[:2] + "****" + card[len(card)-2:]
}

// buildIntegralCards - 在内存中生成一批卡密并做去重（降低与数据库唯一索引冲突的概率）
func buildIntegralCards(batch string, value, count, length int, expireTime int64, remark string) ([]IntegralCard, error) {
	seen := make(map[string]struct{}, count)
	list := make([]IntegralCard, 0, count)

	for len(list) < count {
		card, err := randomIntegralCard(length)
		if err != nil {
			return nil, errors.New("卡密生成失败，请稍后重试！")
		}
		if _, exist := seen[card]; exist {
			continue
		}
		seen[card] = struct{}{}
		list = append(list, IntegralCard{
			Card:       card,
			Value:      value,
			Status:     IntegralCardStatusUnused,
			Batch:      batch,
			ExpireTime: expireTime,
			Remark:     remark,
		})
	}

	return list, nil
}

// GenerateIntegralCards - 批量生成卡密
// value: 积分面额（>0）；count: 生成数量（1~1000）；length: 卡密长度（8~64，默认 16）；
// expireTime: 过期时间戳（0 表示永久有效，>0 时必须晚于当前时间）；remark: 备注
func GenerateIntegralCards(value, count, length int, expireTime int64, remark string) (list []IntegralCard, batch string, err error) {

	if value <= 0 {
		return nil, "", errors.New("积分面额必须大于0！")
	}
	if value > IntegralCardMaxValue {
		return nil, "", fmt.Errorf("积分面额不能超过 %d！", IntegralCardMaxValue)
	}
	if count <= 0 || count > IntegralCardMaxCount {
		return nil, "", fmt.Errorf("生成数量需在 1 ~ %d 之间！", IntegralCardMaxCount)
	}
	if length <= 0 {
		length = IntegralCardDefaultLength
	}
	if length < IntegralCardMinLength || length > IntegralCardMaxLength {
		return nil, "", fmt.Errorf("卡密长度需在 %d ~ %d 之间！", IntegralCardMinLength, IntegralCardMaxLength)
	}
	if expireTime > 0 && expireTime <= time.Now().Unix() {
		return nil, "", errors.New("卡密有效期必须晚于当前时间！")
	}

	batch = fmt.Sprintf("%s%s", time.Now().Format("20060102150405"), randomIntegralBatch())

	// 唯一索引兜底防重：极小概率冲突时重新生成整批
	for attempt := 0; attempt < 3; attempt++ {
		list, err = buildIntegralCards(batch, value, count, length, expireTime, remark)
		if err != nil {
			return nil, "", err
		}
		if err = facade.DB.Drive().Create(&list).Error; err == nil {
			return list, batch, nil
		}
		if !strings.Contains(err.Error(), "Duplicate") {
			break
		}
	}

	facade.Log.Error(map[string]any{
		"error":  err,
		"value":  value,
		"count":  count,
		"length": length,
	}, "批量生成卡密失败")

	return nil, "", errors.New("卡密生成失败，请稍后重试！")
}

// RedeemIntegralCard - 用户兑换卡密（事务：校验卡密 → 原子占用 → 增加积分 → 写流水）
// 通过「状态条件更新 + 影响行数判断」保证同一张卡密在并发下也只能被兑换一次
func RedeemIntegralCard(uid int, card string) (result facade.H, err error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	card = normalizeIntegralCard(card)
	if utils.Is.Empty(card) {
		return nil, errors.New("卡密不能为空！")
	}
	if len(card) < IntegralCardMinLength || len(card) > IntegralCardMaxLength {
		return nil, errors.New("卡密格式不正确！")
	}

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 1. 查询卡密
		var record IntegralCard
		if err := tx.Where("card = ?", card).First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("卡密不存在！")
			}
			return err
		}

		// 2. 校验使用状态与有效期
		if record.Status == IntegralCardStatusUsed {
			return errors.New("卡密已被使用！")
		}

		now := time.Now().Unix()
		if record.ExpireTime > 0 && now > record.ExpireTime {
			return errors.New("卡密已过期！")
		}

		// 3. 原子占用卡密：仅「未使用」状态可被占用
		occupied := tx.Model(&IntegralCard{}).
			Where("id = ? AND status = ?", record.Id, IntegralCardStatusUnused).
			Updates(map[string]any{
				"status":   IntegralCardStatusUsed,
				"uid":      uid,
				"use_time": now,
			})
		if occupied.Error != nil {
			return occupied.Error
		}
		if occupied.RowsAffected == 0 {
			return errors.New("卡密已被使用！")
		}

		// 4. 增加用户积分余额（数据库自增，避免并发覆盖）
		if err := tx.Model(&Users{}).Where("id = ?", uid).
			UpdateColumn("integral", gorm.Expr("integral + ?", record.Value)).Error; err != nil {
			return err
		}

		// 5. 读取最新余额，用于流水快照对账
		var user Users
		if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
			return errors.New("用户不存在！")
		}

		// 6. 写入积分流水（卡密兑换类型）
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       record.Value,
			Type:        IntegralTypeCard,
			Description: "卡密兑换积分",
			Json: utils.Json.Encode(map[string]any{
				"card_id":       record.Id,
				"card":          maskIntegralCard(card),
				"batch":         record.Batch,
				"balance_after": user.Integral,
			}),
		}).Error; err != nil {
			return err
		}

		result = facade.H{
			"card_id":  record.Id,
			"value":    record.Value,
			"batch":    record.Batch,
			"integral": user.Integral,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	facade.Log.Info(map[string]any{
		"uid":     uid,
		"card_id": result["card_id"],
		"value":   result["value"],
	}, "卡密兑换积分成功")

	return result, nil
}
