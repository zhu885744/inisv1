package model

import (
	"errors"
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

/**
 * UserBlocks - 用户拉黑关系表
 *
 * 只存一条「A 拉黑了 B」的记录，但语义上是**双向**的：
 *   - A 的内容不再出现在 B 的聚合流里，B 的内容也不再出现在 A 的流里（Feed 已接入，见 feed.go）
 *   - 双方不能互相发私信（私信上线后由 CanInteract 前置校验）
 *   - 拉黑后是否互不可见评论 / 主页，留给后续批次按需接入
 *     （接入点统一走 IsBlocked / BlockedUids，不要在各处自己写 SQL）
 *
 * 为什么用物理删除而不是 status 软开关：拉黑 / 解除是低频操作，
 * 保留历史会带来「曾经的拉黑关系影响现在判断」的坑（对比 user_follows 保留 status=0 是为了关注历史）。
 */
type UserBlocks struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); index:idx_user_blocks_uid; uniqueIndex:uk_user_blocks_pair,priority:1; comment:发起拉黑的用户ID;" json:"uid"`
	BlockUid int    `gorm:"type:int(32); index:idx_user_blocks_block_uid; uniqueIndex:uk_user_blocks_pair,priority:2; comment:被拉黑的用户ID;" json:"block_uid"`
	Reason   string `gorm:"size:256; comment:拉黑原因（仅自己可见）; default:Null;" json:"reason"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitUserBlocks() {
	if err := facade.DB.Drive().AutoMigrate(&UserBlocks{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserBlocks表迁移失败")
		return
	}
}

// Block - 拉黑（幂等：重复拉黑不报错，也不产生第二条记录）
//
// 要点：uk_user_blocks_pair 是 (uid, block_uid) 的唯一索引，**不含 delete_time**，
// 所以历史软删除记录仍会占着这个唯一键 —— 必须「先找回（含回收站）→ 恢复 → 更新原因」，
// 否则直接 Create 会撞 1062 Duplicate entry。
func (this *UserBlocks) Block(uid, blockUid int, reason string) error {

	if uid <= 0 || blockUid <= 0 {
		return errors.New("参数不完整！")
	}
	if uid == blockUid {
		return errors.New("不能拉黑自己！")
	}

	// 已有记录（生效中或已在回收站）：恢复并更新原因，幂等返回
	if id := this.findPairId(uid, blockUid); id > 0 {
		return this.restorePair(id, reason)
	}

	_, err := facade.DB.Model(&UserBlocks{}).Create(&UserBlocks{
		Uid:      uid,
		BlockUid: blockUid,
		Reason:   reason,
	})
	if err == nil {
		return nil
	}

	// 并发（连点、客户端重试）时可能已被另一次请求插入：再找一次，
	// 找到就按「已拉黑」处理 —— 不要把 MySQL 的 1062 原文抛给用户
	if id := this.findPairId(uid, blockUid); id > 0 {
		return this.restorePair(id, reason)
	}

	facade.Log.Error(map[string]any{"error": err, "uid": uid, "block_uid": blockUid}, "拉黑失败")
	return errors.New("拉黑失败，请稍后重试！")
}

// Unblock - 解除拉黑（物理删除，幂等：本就没拉黑时也返回成功）
//
// 为什么用物理删除而不是软删除：唯一索引不含 delete_time，软删的记录会一直占着唯一键，
// 下次拉黑必须走「恢复」分支才不报错；而拉黑关系只影响两个人、没有留史价值，
// 物理删除最干净（也能顺手清掉历史软删数据）。
func (this *UserBlocks) Unblock(uid, blockUid int) error {

	if uid <= 0 || blockUid <= 0 {
		return errors.New("参数不完整！")
	}

	_, err := facade.DB.Model(&UserBlocks{}).Force().Where([]any{
		[]any{"uid", "=", uid},
		[]any{"block_uid", "=", blockUid},
	}).Delete()

	return err
}

// findPairId - 查 (uid, block_uid) 这条记录的 id（**包含回收站里的**；0 表示没有）
//
// 用 Select() 取 map，而不是 Find(&struct)：
// facade 的 Where 是**按列名解析**的（非法列名会直接 return this 丢弃该条件），
// 传 "uid = ? AND block_uid = ?" 这样的原始 SQL 会被当成列名丢掉，查询条件静默失效，
// 结果就是「查不到 → 走新增 → 撞唯一索引」。
func (this *UserBlocks) findPairId(uid, blockUid int) int {

	rows, _ := facade.DB.Model(&[]UserBlocks{}).WithTrashed().Where([]any{
		[]any{"uid", "=", uid},
		[]any{"block_uid", "=", blockUid},
	}).Select()

	if len(rows) == 0 {
		return 0
	}

	return cast.ToInt(rows[0]["id"])
}

// restorePair - 把（可能已在回收站的）记录恢复为生效状态，并更新原因
func (this *UserBlocks) restorePair(id int, reason string) error {

	if _, err := facade.DB.Model(&UserBlocks{}).Restore(id); err != nil {
		return err
	}

	_, err := facade.DB.Model(&UserBlocks{}).Where("id", id).Update(map[string]any{"reason": reason})

	return err
}

// IsBlocked - 我是否拉黑了 TA（只查单向，用于「我是否已拉黑」的界面状态）
func (this *UserBlocks) IsBlocked(uid, blockUid int) bool {
	if uid <= 0 || blockUid <= 0 {
		return false
	}
	exist, _ := facade.DB.Model(&UserBlocks{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"block_uid", "=", blockUid},
	}).Exist()
	return exist
}

// BlockedUids - 与我有拉黑关系的人（**双向**：我拉黑的 + 拉黑我的）
//
// 用于 Feed 过滤与私信前置校验：任何「不要把对方内容/消息给我看」的场景都走这里，
// 保证两个方向的语义一致（自己拉黑的人看不到，拉黑自己的人也不冒出来）。
func (this *UserBlocks) BlockedUids(uid int) []int {

	if uid <= 0 {
		return []int{}
	}

	result := make(map[int]bool)

	// 我拉黑的人
	if column, err := facade.DB.Model(&[]UserBlocks{}).Where("uid", uid).Column("block_uid"); err == nil {
		for _, id := range cast.ToIntSlice(column) {
			if id > 0 {
				result[id] = true
			}
		}
	}

	// 拉黑我的人
	if column, err := facade.DB.Model(&[]UserBlocks{}).Where("block_uid", uid).Column("uid"); err == nil {
		for _, id := range cast.ToIntSlice(column) {
			if id > 0 {
				result[id] = true
			}
		}
	}

	ids := make([]int, 0, len(result))
	for id := range result {
		ids = append(ids, id)
	}

	return ids
}

// CanInteract - 双方是否还能互动（false = 存在任一方向的拉黑）
//
// 私信 / 评论 / @ 等功能上线时统一调用这里，不要在业务里直接查表。
// uid == targetUid 时返回 true（自己与自己不构成拉黑关系）。
func (this *UserBlocks) CanInteract(uid, targetUid int) bool {

	if uid <= 0 || targetUid <= 0 {
		return false
	}
	if uid == targetUid {
		return true
	}

	exist, _ := facade.DB.Model(&UserBlocks{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"block_uid", "=", targetUid},
	}).Exist()
	if exist {
		return false
	}

	reverse, _ := facade.DB.Model(&UserBlocks{}).Where([]any{
		[]any{"uid", "=", targetUid},
		[]any{"block_uid", "=", uid},
	}).Exist()

	return !reverse
}

// GetBlocked - 我的拉黑列表（分页）
//
// 只返回我**主动拉黑**的人（不含拉黑我的人）：把「谁拉黑了我」暴露给用户会造成额外冲突，
// 而 Feed / 私信的过滤是双向的，功能上并不依赖这个列表。
func (this *UserBlocks) GetBlocked(uid int, page, limit int) ([]map[string]any, int64) {

	query := facade.DB.Model(&[]UserBlocks{}).Where("uid", uid)

	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("create_time desc").Select()

	return data, count
}

// BlockedCount - 我主动拉黑的人数
func (this *UserBlocks) BlockedCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserBlocks{}).Where("uid", uid).Count()
	return count
}

// AfterFind - 查询Hook
func (this *UserBlocks) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
