package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"inis/app/facade"

	"github.com/spf13/cast"
	"gorm.io/gorm/clause"
)

/**
 * 群聊：用户自建的聊天室（多房间）
 *
 * 与公共大厅（id=1）共用 chat_rooms / chat_room_messages 两张表：
 *   - 大厅是「登录即成员」，不写 chat_room_members，靠 PublicRoomId 直接放行；
 *   - 自建群靠 chat_room_members 维护成员与角色（群主 / 管理员 / 成员）；
 *   - 「需审批」的群用 chat_room_join_requests 走申请流程。
 *
 * 这里只做数据与规则，任何 socket 推送都不在此处发生：
 * model 不能反向 import socket（socket 已经 import model），
 * 推送统一由 API 控制器在落库成功后调用 socketctrl 完成。
 */

// ---------- 成员与权限 ----------

// RoomRoleName - 角色中文名（前端展示 / 后台名单用）
func RoomRoleName(role int) string {
	switch role {
	case ChatRoomRoleOwner:
		return "群主"
	case ChatRoomRoleAdmin:
		return "管理员"
	default:
		return "成员"
	}
}

// RoomIsMember - 是否群成员；公共大厅对所有登录用户开放
func RoomIsMember(roomId, uid int) bool {

	if roomId <= 0 || uid <= 0 {
		return false
	}
	if roomId == PublicRoomId {
		return true
	}

	var count int64
	facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("room_id = ? AND uid = ?", roomId, uid).Count(&count)

	return count > 0
}

// RoomRole - 我在群里的角色；非成员返回 -1，公共大厅返回成员（0）
func RoomRole(roomId, uid int) int {

	if roomId <= 0 || uid <= 0 {
		return -1
	}
	if roomId == PublicRoomId {
		return ChatRoomRoleMember
	}

	var row ChatRoomMembers
	if err := facade.DB.Drive().
		Where("room_id = ? AND uid = ?", roomId, uid).First(&row).Error; err != nil {
		return -1
	}

	return row.Role
}

// RoomCanManage - 群主（或超级管理员）：改名 / 公告 / 解散 / 设管理员 / 踢人
func RoomCanManage(roomId, uid int) bool {

	if IsRootAdmin(uid) {
		return true
	}

	return RoomRole(roomId, uid) == ChatRoomRoleOwner
}

// RoomCanModerate - 群主 / 群管理员（或超级管理员）：撤回 / 禁言
func RoomCanModerate(roomId, uid int) bool {

	if IsRootAdmin(uid) {
		return true
	}

	role := RoomRole(roomId, uid)

	return role == ChatRoomRoleOwner || role == ChatRoomRoleAdmin
}

// RoomMemberUids - 群成员 uid 列表；公共大厅返回空（调用方应改为全员广播）
func RoomMemberUids(roomId int) []int {

	if roomId <= 0 || roomId == PublicRoomId {
		return []int{}
	}

	uids := make([]int, 0, 64)
	facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("room_id = ?", roomId).Pluck("uid", &uids)

	return uids
}

// refreshMemberCount - 重算成员数（入群 / 退群 / 踢人 / 审批通过后调用）
//
// 用 COUNT 重算而不是 +1/-1：并发入群时自增会漂移，
// 而列表页展示的人数差一个人没人会在意，漂移了反而难查。
func refreshMemberCount(roomId int) {

	if roomId <= 0 || roomId == PublicRoomId {
		return
	}

	var count int64
	facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("room_id = ?", roomId).Count(&count)

	facade.DB.Drive().Model(&ChatRooms{}).
		Where("id = ?", roomId).Update("member_count", count)
}

// ---------- 建群门槛 ----------

// RoomCanCreate - 是否允许建群，返回 (是否允许, 原因)
//
// 门槛是「或」关系：注册满 RoomCreateMinDays 天，或经验值达到 RoomCreateMinExp。
// 超级管理员不受限（后台排障时不该被自己的规则挡住）。
func RoomCanCreate(uid int) (bool, string) {

	if uid <= 0 {
		return false, "请先登录！"
	}
	if IsRootAdmin(uid) {
		return true, ""
	}

	var user Users
	if err := facade.DB.Drive().Where("id = ?", uid).First(&user).Error; err != nil || user.Id <= 0 {
		return false, "用户不存在！"
	}
	if user.Status != UserStatusNormal {
		return false, "账号状态异常，暂时不能创建聊天室！"
	}

	var owned int64
	facade.DB.Drive().Model(&ChatRooms{}).
		Where("owner_uid = ?", uid).Count(&owned)
	if owned >= ChatRoomCreateMaxOwned {
		return false, fmt.Sprintf("每人最多创建 %d 个聊天室！", ChatRoomCreateMaxOwned)
	}

	if user.Exp >= ChatRoomCreateMinExp {
		return true, ""
	}

	if user.CreateTime > 0 {
		days := (time.Now().Unix() - user.CreateTime) / 86400
		if days >= ChatRoomCreateMinDays {
			return true, ""
		}
		return false, fmt.Sprintf("注册满 %d 天或经验达到 %d 才能创建聊天室！", ChatRoomCreateMinDays, ChatRoomCreateMinExp)
	}

	return false, fmt.Sprintf("经验达到 %d 才能创建聊天室！", ChatRoomCreateMinExp)
}

// ---------- 房间读写 ----------

// RoomCreate - 创建群聊，创建者自动成为群主
func RoomCreate(ownerUid int, params map[string]any) (map[string]any, error) {

	if ok, reason := RoomCanCreate(ownerUid); !ok {
		return nil, errors.New(reason)
	}

	name := strings.TrimSpace(cast.ToString(params["name"]))
	if len([]rune(name)) < 2 || len([]rune(name)) > 30 {
		return nil, errors.New("聊天室名称需要 2 ~ 30 个字！")
	}

	description := strings.TrimSpace(cast.ToString(params["description"]))
	if len([]rune(description)) > 200 {
		return nil, errors.New("聊天室简介不能超过 200 个字！")
	}

	avatar := strings.TrimSpace(cast.ToString(params["avatar"]))
	if avatar != "" && !isRoomMediaUrl(avatar) {
		return nil, errors.New("头像地址不合法！")
	}

	joinType := cast.ToInt(params["join_type"])
	if joinType < ChatRoomJoinFree || joinType > ChatRoomJoinInvite {
		return nil, errors.New("入群方式不正确！")
	}

	room := ChatRooms{
		Name:        name,
		Description: description,
		Avatar:      avatar,
		Status:      1,
		OwnerUid:    ownerUid,
		JoinType:    joinType,
		MemberCount: 1,
		MaxMembers:  ChatRoomMaxMembers,
	}

	if err := facade.DB.Drive().Create(&room).Error; err != nil {
		return nil, errors.New("创建失败，请稍后重试！")
	}

	member := ChatRoomMembers{
		RoomId:   room.Id,
		Uid:      ownerUid,
		Role:     ChatRoomRoleOwner,
		JoinTime: time.Now().Unix(),
	}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err, "room_id": room.Id}, "群主成员行写入失败")
	}

	return chatRoomItem(&room), nil
}

// RoomList - 房间列表
//
// scope：
//   - mine     我加入的群（不含大厅，大厅由前端固定入口展示）
//   - discover 可发现并加入的群（不含「仅邀请」、不含我已在的）
//   - all      全部（后台管理用）
func RoomList(uid int, scope, keyword string) []map[string]any {

	result := make([]map[string]any, 0)
	scope = strings.ToLower(strings.TrimSpace(scope))

	db := facade.DB.Drive().Model(&ChatRooms{})

	var myRooms []ChatRoomMembers
	if uid > 0 {
		facade.DB.Drive().Model(&ChatRoomMembers{}).Where("uid = ?", uid).Find(&myRooms)
	}
	joined := make(map[int]int, len(myRooms))
	for _, item := range myRooms {
		joined[item.RoomId] = item.Role
	}

	switch scope {
	case "mine":
		if uid <= 0 {
			return result
		}
		ids := make([]int, 0, len(myRooms))
		for _, item := range myRooms {
			ids = append(ids, item.RoomId)
		}
		if len(ids) == 0 {
			return result
		}
		db = db.Where("id IN ?", ids)
	case "discover":
		// 大厅是全员隐式成员，前端固定入口展示，不在「发现」里重复出现
		db = db.Where("id <> ? AND status = 1 AND join_type <> ?", PublicRoomId, ChatRoomJoinInvite)
		if len(joined) > 0 {
			ids := make([]int, 0, len(joined))
			for id := range joined {
				ids = append(ids, id)
			}
			db = db.Where("id NOT IN ?", ids)
		}
	}

	if keyword = strings.TrimSpace(keyword); keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}

	var rooms []ChatRooms
	if err := db.Order("last_msg_id desc, id asc").Limit(50).Find(&rooms).Error; err != nil {
		return result
	}

	if len(rooms) == 0 {
		return result
	}

	// 最后一条消息的摘要：批量查，避免每行一次 SQL
	lastIds := make([]int, 0, len(rooms))
	for _, room := range rooms {
		if room.LastMsgId > 0 {
			lastIds = append(lastIds, room.LastMsgId)
		}
	}
	briefs := make(map[int]map[string]any, len(lastIds))
	if len(lastIds) > 0 {
		var rows []ChatRoomMessages
		facade.DB.Drive().Where("id IN ?", lastIds).Find(&rows)
		for i := range rows {
			briefs[rows[i].Id] = roomLastMessage(&rows[i])
		}
	}

	for i := range rooms {
		item := chatRoomItem(&rooms[i])
		role, ok := joined[rooms[i].Id]
		item["joined"] = ok
		item["my_role"] = -1
		if ok {
			item["my_role"] = role
		}
		item["last_message"] = briefs[rooms[i].LastMsgId]
		item["unread"] = roomUnread(rooms[i].Id, uid)
		result = append(result, item)
	}

	return result
}

// roomLastMessage - 列表页的最后一条消息摘要
func roomLastMessage(item *ChatRoomMessages) map[string]any {

	if item == nil || item.Id <= 0 {
		return nil
	}

	return map[string]any{
		"id":          item.Id,
		"uid":         item.Uid,
		"msg_type":    item.MsgType,
		"content":     ChatPreview(item.MsgType, item.Content),
		"is_revoked":  item.IsRevoked,
		"create_time": item.CreateTime,
	}
}

// roomUnread - 我的未读数（大厅不统计；不是成员返回 0）
func roomUnread(roomId, uid int) int {

	if roomId <= 0 || roomId == PublicRoomId || uid <= 0 {
		return 0
	}

	var member ChatRoomMembers
	if err := facade.DB.Drive().
		Where("room_id = ? AND uid = ?", roomId, uid).First(&member).Error; err != nil {
		return 0
	}

	if member.LastReadId <= 0 {
		// 从没点开过：只算「我入群之后」的消息，别把入群前几百条都算成未读
		var count int64
		facade.DB.Drive().Model(&ChatRoomMessages{}).
			Where("room_id = ? AND create_time >= ?", roomId, member.JoinTime).Count(&count)
		return int(count)
	}

	var count int64
	facade.DB.Drive().Model(&ChatRoomMessages{}).
		Where("room_id = ? AND id > ?", roomId, member.LastReadId).Count(&count)

	return int(count)
}

// RoomDetail - 房间详情 + 我的状态
func RoomDetail(roomId, uid int) (map[string]any, error) {

	if roomId <= 0 {
		return nil, errors.New("聊天室不存在！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
		return nil, errors.New("聊天室不存在或已解散！")
	}

	role := -1
	joined := roomId == PublicRoomId
	lastReadId := 0
	muteNotify := 0

	if roomId != PublicRoomId && uid > 0 {
		var member ChatRoomMembers
		if err := facade.DB.Drive().
			Where("room_id = ? AND uid = ?", roomId, uid).First(&member).Error; err == nil {
			joined = true
			role = member.Role
			lastReadId = member.LastReadId
			muteNotify = member.MuteNotify
		} else if IsRootAdmin(uid) {
			// 超级管理员进任意群都能看（后台审消息 / 排障），但不写成员行
			joined = true
			role = ChatRoomRoleMember
		}
	}

	item := chatRoomItem(&room)
	item["joined"] = joined
	item["my_role"] = role
	item["last_read_id"] = lastReadId
	item["mute_notify"] = muteNotify
	item["unread"] = roomUnread(roomId, uid)
	item["is_root_admin"] = IsRootAdmin(uid)

	return item, nil
}

// RoomJoin - 入群
//
// 返回值 joined/pending 二选一：
//   - joined  = true  → 已经是 / 刚成为成员
//   - pending = true  → 已提交申请，等群主或管理员审批
func RoomJoin(roomId, uid int, message string) (map[string]any, error) {

	if roomId <= 0 || uid <= 0 {
		return nil, errors.New("参数不完整！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
		return nil, errors.New("聊天室不存在或已解散！")
	}
	if room.Status != 1 {
		return nil, errors.New("聊天室已关闭，暂时不能加入！")
	}

	if roomId == PublicRoomId {
		return map[string]any{"joined": true, "pending": false, "room": chatRoomItem(&room)}, nil
	}

	if RoomIsMember(roomId, uid) {
		return map[string]any{"joined": true, "pending": false, "room": chatRoomItem(&room)}, nil
	}

	if room.MaxMembers > 0 && room.MemberCount >= room.MaxMembers && room.JoinType == ChatRoomJoinFree {
		return nil, errors.New("聊天室人数已满！")
	}

	switch room.JoinType {
	case ChatRoomJoinFree:
		member := ChatRoomMembers{
			RoomId:   roomId,
			Uid:      uid,
			Role:     ChatRoomRoleMember,
			JoinTime: time.Now().Unix(),
		}
		if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
			return nil, errors.New("入群失败，请稍后重试！")
		}
		refreshMemberCount(roomId)
		return map[string]any{"joined": true, "pending": false, "room": (&ChatRooms{}).Info(roomId)}, nil

	case ChatRoomJoinApprove:
		// 已有待审申请就不重复插入，避免群主看到一屏同一个人
		var exist ChatRoomJoinRequests
		if err := facade.DB.Drive().
			Where("room_id = ? AND uid = ? AND status = ?", roomId, uid, ChatRoomRequestPending).
			First(&exist).Error; err == nil && exist.Id > 0 {
			return map[string]any{"joined": false, "pending": true, "room": chatRoomItem(&room)}, nil
		}

		request := ChatRoomJoinRequests{
			RoomId:  roomId,
			Uid:     uid,
			Status:  ChatRoomRequestPending,
			Message: truncateRunes(strings.TrimSpace(message), 120),
		}
		if err := facade.DB.Drive().Create(&request).Error; err != nil {
			return nil, errors.New("申请提交失败，请稍后重试！")
		}
		return map[string]any{"joined": false, "pending": true, "room": chatRoomItem(&room)}, nil
	}

	return nil, errors.New("该聊天室仅限邀请加入！")
}

// RoomLeave - 退群（群主不能直接退，需先解散）
func RoomLeave(roomId, uid int) error {

	if roomId <= 0 || uid <= 0 {
		return errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return errors.New("公共聊天室不能退出！")
	}

	role := RoomRole(roomId, uid)
	if role < 0 {
		return errors.New("你还不是该聊天室的成员！")
	}
	if role == ChatRoomRoleOwner {
		return errors.New("群主不能退出，请先解散聊天室！")
	}

	if err := facade.DB.Drive().Unscoped().
		Where("room_id = ? AND uid = ?", roomId, uid).
		Delete(&ChatRoomMembers{}).Error; err != nil {
		return errors.New("退出失败，请稍后重试！")
	}

	// 成员的禁言记录一并清掉：人都不在群里了，名单里留着没有意义
	facade.DB.Drive().Unscoped().
		Where("room_id = ? AND uid = ?", roomId, uid).
		Delete(&ChatRoomMutes{})

	refreshMemberCount(roomId)

	return nil
}

// RoomDissolve - 解散群聊（群主 / 超级管理员；大厅不可解散）
//
// 房间软删除、成员与申请物理删除；消息保留（后台还能查历史），
// 但前台因为房间查不到而无法再进入。
func RoomDissolve(roomId, uid int) error {

	if roomId <= 0 {
		return errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return errors.New("公共聊天室不能解散！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
		return errors.New("聊天室不存在或已解散！")
	}

	if !RoomCanManage(roomId, uid) {
		return errors.New("只有群主可以解散聊天室！")
	}

	db := facade.DB.Drive()

	if err := db.Delete(&ChatRooms{}, roomId).Error; err != nil {
		return errors.New("解散失败，请稍后重试！")
	}

	db.Unscoped().Where("room_id = ?", roomId).Delete(&ChatRoomMembers{})
	db.Unscoped().Where("room_id = ?", roomId).Delete(&ChatRoomJoinRequests{})
	db.Unscoped().Where("room_id = ?", roomId).Delete(&ChatRoomMutes{})

	return nil
}

// RoomMembers - 成员列表（群主 / 管理员排前面）
func RoomMembers(roomId int, limit int) []map[string]any {

	result := make([]map[string]any, 0)
	if roomId <= 0 || roomId == PublicRoomId {
		return result
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	var rows []ChatRoomMembers
	facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("room_id = ?", roomId).
		Order("role desc, id asc").Limit(limit).Find(&rows)

	for _, row := range rows {
		result = append(result, map[string]any{
			"uid":          row.Uid,
			"role":         row.Role,
			"role_name":    RoomRoleName(row.Role),
			"mute_notify":  row.MuteNotify,
			"last_read_id": row.LastReadId,
			"join_time":    row.JoinTime,
		})
	}

	return result
}

// RoomSetRole - 设置 / 撤销群管理员（只有群主或超级管理员能操作）
func RoomSetRole(roomId, operatorUid, targetUid, role int) error {

	if roomId <= 0 || targetUid <= 0 {
		return errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return errors.New("公共聊天室不需要设置管理员！")
	}
	if !RoomCanManage(roomId, operatorUid) {
		return errors.New("只有群主可以设置管理员！")
	}
	if role != ChatRoomRoleMember && role != ChatRoomRoleAdmin {
		return errors.New("只能设置为成员或管理员！")
	}
	if targetUid == operatorUid {
		return errors.New("不能修改自己的角色！")
	}

	var member ChatRoomMembers
	if err := facade.DB.Drive().
		Where("room_id = ? AND uid = ?", roomId, targetUid).First(&member).Error; err != nil {
		return errors.New("该用户不在聊天室里！")
	}
	if member.Role == ChatRoomRoleOwner {
		return errors.New("不能修改群主的角色！")
	}

	return facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("id = ?", member.Id).Update("role", role).Error
}

// RoomKick - 移出成员（群主 / 管理员；管理员不能踢管理员与群主）
func RoomKick(roomId, operatorUid, targetUid int) error {

	if roomId <= 0 || targetUid <= 0 {
		return errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return errors.New("公共聊天室不能移除成员！")
	}
	if !RoomCanModerate(roomId, operatorUid) {
		return errors.New("没有权限移除成员！")
	}
	if targetUid == operatorUid {
		return errors.New("不能移除自己！")
	}

	targetRole := RoomRole(roomId, targetUid)
	if targetRole < 0 {
		return errors.New("该用户不在聊天室里！")
	}
	if targetRole == ChatRoomRoleOwner {
		return errors.New("不能移除群主！")
	}
	// 管理员只能踢普通成员：同级互踢会让群管理变成斗殴现场
	if targetRole == ChatRoomRoleAdmin && !IsRootAdmin(operatorUid) &&
		RoomRole(roomId, operatorUid) != ChatRoomRoleOwner {
		return errors.New("管理员不能移除其他管理员！")
	}

	return RoomLeave(roomId, targetUid)
}

// RoomAddMember - 群主 / 管理员直接拉人（「仅邀请」的群只能这样进人）
//
// 与 RoomJoin 的区别：这里不看 join_type，也不需要对方申请；
// 但同样受人数上限约束，且目标必须真实存在。
func RoomAddMember(roomId, operatorUid, targetUid int) error {

	if roomId <= 0 || targetUid <= 0 {
		return errors.New("参数不完整！")
	}
	if roomId == PublicRoomId {
		return errors.New("公共大厅不需要邀请！")
	}
	if !RoomCanModerate(roomId, operatorUid) {
		return errors.New("没有权限邀请成员！")
	}
	if targetUid == operatorUid {
		return errors.New("不能邀请自己！")
	}

	var user Users
	if err := facade.DB.Drive().Select("id", "status").Where("id = ?", targetUid).First(&user).Error; err != nil || user.Id <= 0 {
		return errors.New("用户不存在！")
	}
	if user.Status != UserStatusNormal {
		return errors.New("该账号状态异常，暂时不能邀请！")
	}

	if RoomIsMember(roomId, targetUid) {
		return errors.New("该用户已经在聊天室里了！")
	}

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
		return errors.New("聊天室不存在或已解散！")
	}
	if room.MaxMembers > 0 && room.MemberCount >= room.MaxMembers {
		return errors.New("聊天室人数已满！")
	}

	member := ChatRoomMembers{
		RoomId:   roomId,
		Uid:      targetUid,
		Role:     ChatRoomRoleMember,
		JoinTime: time.Now().Unix(),
	}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
		return errors.New("邀请失败，请稍后重试！")
	}

	refreshMemberCount(roomId)

	return nil
}

// RoomRequests - 待审的入群申请
func RoomRequests(roomId int) []map[string]any {

	result := make([]map[string]any, 0)
	if roomId <= 0 || roomId == PublicRoomId {
		return result
	}

	var rows []ChatRoomJoinRequests
	facade.DB.Drive().Model(&ChatRoomJoinRequests{}).
		Where("room_id = ? AND status = ?", roomId, ChatRoomRequestPending).
		Order("id asc").Limit(100).Find(&rows)

	for _, row := range rows {
		result = append(result, map[string]any{
			"id":          row.Id,
			"uid":         row.Uid,
			"message":     row.Message,
			"status":      row.Status,
			"create_time": row.CreateTime,
		})
	}

	return result
}

// RoomHandleRequest - 审批入群申请
func RoomHandleRequest(roomId, operatorUid, requestId int, approve bool) error {

	if roomId <= 0 || requestId <= 0 {
		return errors.New("参数不完整！")
	}
	if !RoomCanModerate(roomId, operatorUid) {
		return errors.New("没有权限审批入群申请！")
	}

	var request ChatRoomJoinRequests
	if err := facade.DB.Drive().
		Where("id = ? AND room_id = ?", requestId, roomId).First(&request).Error; err != nil {
		return errors.New("申请不存在！")
	}
	if request.Status != ChatRoomRequestPending {
		return errors.New("该申请已处理！")
	}

	status := ChatRoomRequestRejected
	if approve {
		status = ChatRoomRequestApproved
	}

	// 通过前先校验容量：先把申请标成「已通过」再发现人满了，
	// 申请就永远停在通过状态却进不了群，群里也看不到待处理项了。
	if approve {
		var room ChatRooms
		if err := facade.DB.Drive().Where("id = ?", roomId).First(&room).Error; err != nil || room.Id <= 0 {
			return errors.New("聊天室不存在或已解散！")
		}
		if room.MaxMembers > 0 && room.MemberCount >= room.MaxMembers {
			return errors.New("聊天室人数已满，无法通过申请！")
		}
	}

	if err := facade.DB.Drive().Model(&ChatRoomJoinRequests{}).
		Where("id = ?", request.Id).Updates(map[string]any{
		"status":       status,
		"operator_uid": operatorUid,
		"operate_time": time.Now().Unix(),
	}).Error; err != nil {
		return errors.New("处理失败，请稍后重试！")
	}

	if !approve {
		return nil
	}

	member := ChatRoomMembers{
		RoomId:   roomId,
		Uid:      request.Uid,
		Role:     ChatRoomRoleMember,
		JoinTime: time.Now().Unix(),
	}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
		return errors.New("成员写入失败，请稍后重试！")
	}

	refreshMemberCount(roomId)

	return nil
}

// RoomMarkRead - 记录已读水位（进群 / 拉到最新消息后调用）
func RoomMarkRead(roomId, uid, lastId int) {

	if roomId <= 0 || roomId == PublicRoomId || uid <= 0 || lastId <= 0 {
		return
	}

	facade.DB.Drive().Model(&ChatRoomMembers{}).
		Where("room_id = ? AND uid = ? AND last_read_id < ?", roomId, uid, lastId).
		Update("last_read_id", lastId)
}

// RoomCountByOwner - 某人拥有的群数（后台 / 个人中心展示用）
func RoomCountByOwner(uid int) int64 {

	var count int64
	facade.DB.Drive().Model(&ChatRooms{}).Where("owner_uid = ?", uid).Count(&count)

	return count
}

// RoomListAll - 后台：全部房间（含官方大厅）
func RoomListAll(keyword string, page, size int) ([]map[string]any, int64) {

	db := facade.DB.Drive().Model(&ChatRooms{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	db.Count(&total)

	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	var rooms []ChatRooms
	db.Order("id asc").Offset((page - 1) * size).Limit(size).Find(&rooms)

	// 群主信息：后台列表要显示「谁建的群」，批量查一次避免 N+1
	ownerIds := make([]int, 0, len(rooms))
	for i := range rooms {
		if rooms[i].OwnerUid > 0 {
			ownerIds = append(ownerIds, rooms[i].OwnerUid)
		}
	}
	owners := make(map[int]map[string]any, len(ownerIds))
	if len(ownerIds) > 0 {
		var users []Users
		facade.DB.Drive().Select("id", "nickname", "avatar").
			Where("id IN ?", ownerIds).Find(&users)
		for i := range users {
			owners[users[i].Id] = map[string]any{
				"id":       users[i].Id,
				"nickname": users[i].Nickname,
				"avatar":   users[i].Avatar,
			}
		}
	}

	result := make([]map[string]any, 0, len(rooms))
	for i := range rooms {
		item := chatRoomItem(&rooms[i])
		item["owner"] = owners[rooms[i].OwnerUid]
		item["last_message"] = roomLastMessageById(rooms[i].LastMsgId)
		result = append(result, item)
	}

	return result, total
}

// roomLastMessageById - 按消息 id 取摘要
func roomLastMessageById(id int) map[string]any {

	if id <= 0 {
		return nil
	}

	var row ChatRoomMessages
	if err := facade.DB.Drive().Where("id = ?", id).First(&row).Error; err != nil {
		return nil
	}

	return roomLastMessage(&row)
}


