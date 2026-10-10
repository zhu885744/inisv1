package model

import (
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type ApiKeys struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Value 	string `gorm:"comment:值; default:Null;" json:"value"`
	Remark  string `gorm:"comment:备注; default:Null;" json:"remark"`
	// Status 启用状态：0=停用（保留记录但拒绝调用 → 403 该密钥已停用） 1=启用
	//
	// 与「删除」的区别：停用是「可逆的临时关闭」，密钥值不变、记录还在，
	// 适合临时切断某个调用方（排查问题 / 欠费 / 违规），恢复时改回启用即可。
	Status int `gorm:"tinyint; index; comment:状态（0停用 1启用）; default:1;" json:"status"`
	// ExpireTime 有效期（unix 秒，0=永久）；超过后调用返回 403 该密钥已过期
	ExpireTime int64 `gorm:"comment:有效期（0=永久）; default:0;" json:"expire_time"`
	// LastUsedAt 最后使用时间（unix 秒，0=从未使用）
	LastUsedAt int64 `gorm:"comment:最后使用时间; default:0;" json:"last_used_at"`
	// UseCount 累计调用次数（校验通过时自增，用于用量审计）
	UseCount int `gorm:"type:int(32); comment:累计调用次数; default:0;" json:"use_count"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// 密钥启用状态
const (
	ApiKeyStatusOff = 0 // 停用
	ApiKeyStatusOn  = 1 // 启用
)

// Usable - 密钥当前是否可用（启用中且未过期）
//
// 校验口径（与 app/api/middleware/api-key.go 一致）：
//   - status != 1 → 停用；
//   - expire_time > 0 且已过 → 过期；
//   - expire_time = 0 → 永久有效。
func (this ApiKeys) Usable(now int64) bool {
	if this.Status != ApiKeyStatusOn {
		return false
	}
	if this.ExpireTime > 0 && now > this.ExpireTime {
		return false
	}
	return true
}

// ApiKeyTouch - 记录一次「校验通过」的调用（累计次数 + 最后使用时间）
//
// 由中间件在校验通过后异步调用：单条 UPDATE 原子自增，避免并发下的读改写丢计数；
// 任何异常只记日志 —— 用量统计不能影响正常请求。
func ApiKeyTouch(id int) {
	if id <= 0 {
		return
	}

	if err := facade.DB.Drive().Exec(
		"UPDATE inis_api_keys SET use_count = use_count + 1, last_used_at = ? "+
			"WHERE id = ? AND (delete_time IS NULL OR delete_time = 0)",
		time.Now().Unix(), id,
	).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "id": id}, "记录密钥使用次数失败")
	}
}

// InitApiKeys - 初始化ApiKeys表
func InitApiKeys() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&ApiKeys{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ApiKeys表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *ApiKeys) AfterFind(tx *gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}