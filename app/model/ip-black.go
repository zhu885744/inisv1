package model

import (
	"fmt"
	"time"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"inis/app/facade"
)

// 封禁等级常量
const (
	BanLevel1 = 1 // 一级封禁：1小时
	BanLevel2 = 2 // 二级封禁：24小时
	BanLevel3 = 3 // 三级封禁：7天
	BanLevel4 = 4 // 四级封禁：永久
)

// 封禁时长映射（小时）
var BanDuration = map[int]int64{
	BanLevel1: 1,                    // 1小时
	BanLevel2: 24,                   // 24小时
	BanLevel3: 24 * 7,               // 7天
	BanLevel4: -1,                   // 永久
}

type IpBlack struct {
	Id         int     				 `gorm:"type:int(32); comment:主键;" json:"id"`
	Ip         string  				 `gorm:"comment:IP; default:Null;" json:"ip"`
	Level      int    				 `gorm:"type:int(8); comment:封禁等级 1-4级; default:1;" json:"level"`
	Duration   int64   				 `gorm:"type:int(64); comment:封禁时长(小时); default:1;" json:"duration"`
	ExpireTime int64   				 `gorm:"type:int(64); comment:解封时间戳; default:0;" json:"expire_time"`
	IsPermanent bool   				 `gorm:"type:tinyint(1); comment:是否永久封禁; default:0;" json:"is_permanent"`
	ViolationCount int 				 `gorm:"type:int(32); comment:累计违规次数; default:1;" json:"violation_count"`
	Agent      string  				 `gorm:"type:varchar(512); comment:浏览器信息; default:Null;" json:"agent"`
	Cause	   string  				 `gorm:"comment:原因; default:Null;" json:"cause"`
	Remark     string 				 `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitIpBlack - 初始化IpBlack表
func InitIpBlack() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&IpBlack{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "IpBlack表迁移失败")
		return
	}
}

// BeforeCreate - 创建前的Hook
func (this *IpBlack) BeforeCreate(tx *gorm.DB) (err error) {

	exist, _ := facade.DB.Model(&IpBlack{}).Where("ip", this.Ip).Exist()
	if exist {
		return fmt.Errorf("ip: %s 已存在", this.Ip)
	}

	return
}

// IsBanned - 检查IP是否在封禁期内
func (this *IpBlack) IsBanned() bool {
	if this.IsPermanent {
		return true
	}
	if this.ExpireTime == 0 {
		return false
	}
	return time.Now().Unix() < this.ExpireTime
}

// CalculateExpireTime - 根据等级计算解封时间
func (this *IpBlack) CalculateExpireTime(level int) {
	this.Level = level
	if duration, ok := BanDuration[level]; ok {
		this.Duration = duration
		if duration == -1 {
			this.IsPermanent = true
			this.ExpireTime = 0
		} else {
			this.IsPermanent = false
			this.ExpireTime = time.Now().Unix() + duration*3600
		}
	}
}

// GetNextLevel - 获取下一个封禁等级
func GetNextBanLevel(currentLevel int) int {
	if currentLevel >= BanLevel4 {
		return BanLevel4
	}
	return currentLevel + 1
}