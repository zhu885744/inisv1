package model

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"inis/app/facade"
)

type IpWhite struct {
	Id         int     				 `gorm:"type:int(32); comment:主键;" json:"id"`
	Ip         string  				 `gorm:"comment:IP; default:Null;" json:"ip"`
	Remark     string 				 `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitIpWhite - 初始化IpWhite表
func InitIpWhite() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&IpWhite{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "IpWhite表迁移失败")
		return
	}
}

// BeforeCreate - 创建前的Hook
func (this *IpWhite) BeforeCreate(tx *gorm.DB) (err error) {

	exist, _ := facade.DB.Model(&IpWhite{}).Where("ip", this.Ip).Exist()
	if exist {
		return fmt.Errorf("ip: %s 已存在白名单中", this.Ip)
	}

	return
}
