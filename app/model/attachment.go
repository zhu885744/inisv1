package model

import (
	"inis/app/facade"

	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Attachment struct {
	Id            uint                  `gorm:"type:int(32); primaryKey; autoIncrement; comment:主键;" json:"id"`
	Uuid          string                `gorm:"size:36; unique; comment:唯一标识;" json:"uuid"`
	OriginalName  string                `gorm:"size:256; comment:原始文件名;" json:"original_name"`
	SaveName      string                `gorm:"size:256; comment:存储文件名;" json:"save_name"`
	SavePath      string                `gorm:"comment:存储相对路径;" json:"save_path"`
	FullUrl       string                `gorm:"comment:完整访问URL;" json:"full_url"`
	FileSize      int64                 `gorm:"type:int(64); comment:文件大小（字节）;" json:"file_size"`
	MimeType      string                `gorm:"size:128; comment:MIME类型;" json:"mime_type"`
	FileExt       string                `gorm:"size:32; comment:文件扩展名;" json:"file_ext"`
	StorageDriver string                `gorm:"size:32; comment:存储驱动;" json:"storage_driver"`
	UploaderId    uint                  `gorm:"type:int(32); index; comment:上传者ID;" json:"uploader_id"`
	TargetType    string                `gorm:"size:32; index; comment:关联业务类型;" json:"target_type"`
	TargetId      uint                  `gorm:"type:int(32); index; comment:关联业务ID;" json:"target_id"`
	FileHash      string                `gorm:"size:32; index; comment:文件MD5值;" json:"file_hash"`
	CreateTime    int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime    int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime    soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitAttachment() {
	err := facade.DB.Drive().AutoMigrate(&Attachment{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Attachment表迁移失败")
		return
	}
}

func (this *Attachment) AfterFind(tx *gorm.DB) (err error) {
	this.FullUrl = utils.Replace(this.FullUrl, DomainTemp1())
	return
}

func (this *Attachment) AfterSave(tx *gorm.DB) (err error) {
	go func() {
		fullUrl := utils.Replace(this.FullUrl, DomainTemp2())
		tx.Model(this).UpdateColumn("full_url", fullUrl)
	}()
	return
}

func (this *Attachment) GenerateUUID() string {
	return uuid.New().String()
}

func (this *Attachment) GetByUUID(uuid string) map[string]any {
	return facade.DB.Model(&Attachment{}).Where("uuid", uuid).Find()
}

func (this *Attachment) GetByHash(fileHash string) map[string]any {
	return facade.DB.Model(&Attachment{}).Where("file_hash", fileHash).Where("status", 1).Find()
}

func (this *Attachment) GetByTarget(targetType string, targetId uint) []map[string]any {
	return facade.DB.Model(&[]Attachment{}).Where("target_type", targetType).Where("target_id", targetId).Where("status", 1).Select()
}

func (this *Attachment) GetByUploader(uploaderId uint, page, limit int) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]Attachment{}).Where("uploader_id", uploaderId).Where("status", 1)
	count := query.Count()
	data := query.Limit(limit).Page(page).Order("create_time desc").Select()
	return data, count
}

func (this *Attachment) DeleteByUUID(uuid string, uploaderId uint, isAdmin bool) bool {
	query := facade.DB.Model(&Attachment{}).Where("uuid", uuid)
	if !isAdmin {
		query = query.Where("uploader_id", uploaderId)
	}
	tx := query.Delete()
	return tx.Error == nil
}

func (this *Attachment) ForceDeleteByUUID(uuid string, isAdmin bool) bool {
	query := facade.DB.Model(&Attachment{}).WithTrashed().Where("uuid", uuid)
	if !isAdmin {
		query = query.Where("uploader_id", cast.ToUint(query.Find()["uploader_id"]))
	}
	tx := query.Force().Delete()
	return tx.Error == nil
}

func (this *Attachment) UpdateStatus(uuid string, status int8, isAdmin bool) bool {
	query := facade.DB.Model(&Attachment{}).Where("uuid", uuid)
	if !isAdmin {
		query = query.Where("uploader_id", cast.ToUint(query.Find()["uploader_id"]))
	}
	tx := query.Update(map[string]any{"status": status})
	return tx.Error == nil
}
