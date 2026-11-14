package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"inis/app/facade"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/**
 * 群邀请：链接 / 二维码
 *
 * 一条邀请 = 一个随机 code（不是群 id，避免被猜），前端拼成
 *   <站点>/join/<code>
 * 二维码只是把这个 URL 画出来，所以链接与二维码共用同一份数据。
 *
 * 规则：
 *   - 只有群主 / 管理员能生成（RoomCanModerate）；
 *   - 有效期可选（0 = 永久），次数可选（0 = 不限）；
 *   - 凭邀请进入不受 join_type 限制 —— 「仅邀请」的群正是靠它进人；
 *   - 仍然受群人数上限约束，也不会把已在群里的人重复插入。
 *
 * 安全：code 用 crypto/rand 生成，长度 16，字符集 base62。
 * 不用时间戳 / 群 id 派生：邀请链接会出现在聊天记录里，可猜等于公开可进。
 */

const (
	// ChatRoomInviteCodeLen - 邀请码长度（62^16 的搜索空间，足够抗猜）
	ChatRoomInviteCodeLen = 16

	// ChatRoomInviteMaxDays - 单个邀请最长有效期（按天）
	ChatRoomInviteMaxDays = 365
)

const roomInviteAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// newRoomInviteCode - 生成邀请码
func newRoomInviteCode() string {

	buf := make([]byte, ChatRoomInviteCodeLen)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败极罕见；退回时间戳 + 纳秒，至少不会撞码到无法加入
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	out := make([]byte, ChatRoomInviteCodeLen)
	for i := range buf {
		out[i] = roomInviteAlphabet[int(buf[i])%len(roomInviteAlphabet)]
	}

	return string(out)
}

// RoomCreateInvite - 生成邀请（群主 / 管理员）
//
// days=0 表示永久有效；maxUses=0 表示不限次数。
func RoomCreateInvite(roomId, uid, days, maxUses int) (map[string]any, error) {

	if roomId <= 0 {
		return nil, errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return nil, errors.New("公共大厅不需要邀请！")
	}
	if !RoomCanModerate(roomId, uid) {
		return nil, errors.New("没有权限生成邀请！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
		return nil, errors.New("聊天室不存在或已解散！")
	}

	if days < 0 {
		days = 0
	}
	if days > ChatRoomInviteMaxDays {
		days = ChatRoomInviteMaxDays
	}
	if maxUses < 0 {
		maxUses = 0
	}

	expire := int64(0)
	if days > 0 {
		expire = time.Now().Unix() + int64(days)*86400
	}

	invite := ChatRoomInvites{
		RoomId:     roomId,
		Code:       newRoomInviteCode(),
		CreatorUid: uid,
		ExpireTime: expire,
		MaxUses:    maxUses,
		UsedCount:  0,
		Status:     1,
	}
	if err := facade.DB.Drive().Create(&invite).Error; err != nil {
		return nil, errors.New("生成失败，请稍后重试！")
	}

	return map[string]any{
		"id":          invite.Id,
		"code":        invite.Code,
		"room_id":     roomId,
		"expire_time": invite.ExpireTime,
		"max_uses":    invite.MaxUses,
		"used_count":  0,
		"create_time": invite.CreateTime,
	}, nil
}

// RoomInviteInfo - 落地页展示：这个邀请对应哪个群、我是否已在群里
//
// 不校验登录以外的条件：未过期就允许预览群名 / 成员数，
// 真正加入由 RoomJoinByCode 再校验一次（预览与加入之间可能被人解散）。
func RoomInviteInfo(code string, uid int) (map[string]any, error) {

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("邀请链接不正确！")
	}

	var invite ChatRoomInvites
	if err := facade.DB.Drive().Where("code = ?", code).First(&invite).Error; err != nil || invite.Id <= 0 {
		return nil, errors.New("邀请链接不存在或已失效！")
	}

	expired := invite.Status != 1 || (invite.ExpireTime > 0 && invite.ExpireTime < time.Now().Unix())
	full := invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", invite.RoomId).First(&room).Error; err != nil || room.Id <= 0 {
		return nil, errors.New("聊天室不存在或已解散！")
	}

	joined := RoomIsMember(invite.RoomId, uid)

	// 邀请人信息：让被邀请的人知道是谁拉的
	var inviter map[string]any
	if invite.CreatorUid > 0 {
		var user Users
		if err := facade.DB.Drive().Select("id", "nickname", "avatar").
			Where("id = ?", invite.CreatorUid).First(&user).Error; err == nil && user.Id > 0 {
			inviter = map[string]any{
				"id":       user.Id,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
			}
		}
	}

	return map[string]any{
		"code":        invite.Code,
		"room":        chatRoomItem(&room),
		"joined":      joined,
		"expired":     expired,
		"full":        full,
		"inviter":     inviter,
		"expire_time": invite.ExpireTime,
		"max_uses":    invite.MaxUses,
		"used_count":  invite.UsedCount,
	}, nil
}

// RoomJoinByCode - 凭邀请加入群聊
//
// 与 RoomJoin 的差别：不看 join_type，也不走审批 —— 拿到链接本身就是邀请。
func RoomJoinByCode(code string, uid int) (map[string]any, error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("邀请链接不正确！")
	}

	var invite ChatRoomInvites
	if err := facade.DB.Drive().Where("code = ?", code).First(&invite).Error; err != nil || invite.Id <= 0 {
		return nil, errors.New("邀请链接不存在或已失效！")
	}
	if invite.Status != 1 {
		return nil, errors.New("邀请链接已被停用！")
	}
	if invite.ExpireTime > 0 && invite.ExpireTime < time.Now().Unix() {
		return nil, errors.New("邀请链接已过期！")
	}
	if invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses {
		return nil, errors.New("邀请链接使用次数已用完！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", invite.RoomId).First(&room).Error; err != nil || room.Id <= 0 {
		return nil, errors.New("聊天室不存在或已解散！")
	}
	if room.Status != 1 {
		return nil, errors.New("聊天室已关闭，暂时不能加入！")
	}

	// 已经在群里：直接返回成功，避免「重复点链接」报错
	if RoomIsMember(invite.RoomId, uid) {
		return map[string]any{
			"room_id": invite.RoomId,
			"room":    chatRoomItem(&room),
			"joined":  true,
			"again":   true,
		}, nil
	}

	if room.MaxMembers > 0 && room.MemberCount >= room.MaxMembers {
		return nil, errors.New("聊天室人数已满！")
	}

	member := ChatRoomMembers{
		RoomId:   invite.RoomId,
		Uid:      uid,
		Role:     ChatRoomRoleMember,
		JoinTime: time.Now().Unix(),
	}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
		return nil, errors.New("加入失败，请稍后重试！")
	}

	refreshMemberCount(invite.RoomId)

	// 计数用自增而不是「读出来 +1」：同一链接被多人同时点用，
	// 读改写会丢计数，导致限次失效。
	facade.DB.Drive().Model(&ChatRoomInvites{}).
		Where("id = ?", invite.Id).
		UpdateColumn("used_count", gorm.Expr("used_count + ?", 1))

	return map[string]any{
		"room_id": invite.RoomId,
		"room":    chatRoomItem(&room),
		"joined":  true,
		"again":   false,
	}, nil
}

// RoomInviteList - 群主 / 管理员查看已生成的邀请（管理用，最多 50 条）
func RoomInviteList(roomId, uid int) ([]map[string]any, error) {

	if roomId <= 0 {
		return nil, errors.New("参数不完整！")
	}
	if !RoomCanModerate(roomId, uid) {
		return nil, errors.New("没有权限查看邀请！")
	}

	var rows []ChatRoomInvites
	facade.DB.Drive().Model(&ChatRoomInvites{}).
		Where("room_id = ?", roomId).Order("id desc").Limit(50).Find(&rows)

	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		list = append(list, map[string]any{
			"id":          row.Id,
			"code":        row.Code,
			"creator_uid": row.CreatorUid,
			"expire_time": row.ExpireTime,
			"max_uses":    row.MaxUses,
			"used_count":  row.UsedCount,
			"status":      row.Status,
			"create_time": row.CreateTime,
		})
	}

	return list, nil
}
