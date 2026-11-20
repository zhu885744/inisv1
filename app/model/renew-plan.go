package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"sort"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

// 续期（Renew）
//
// 续期档位是「每个商品自己的」——每枚勋章、每件装扮在自己的 renew 字段里存一份配置：
//
//	{"enabled": true, "options": [
//	   {"days": 30,  "price": 100, "origin_price": 150, "tag": "推荐"},
//	   {"days": 365, "price": 900, "origin_price": 1200, "tag": "最划算"}
//	]}
//
// 为什么不做「全站统一套餐」：勋章有稀有度、装扮有成本和定价，都是逐个定价的，
// 一套公共档位必然错（改价格会波及所有商品）。配置挂在商品上，改一件只影响一件，
// 商品删掉配置也随之消失，后台也不需要维护「档位 ↔ 商品」的关联。
//
// 下单只传 days：价格由服务端从商品自己的档位里取，前端传不了价，改价也不怕抓包。

// renewMaxYears - 单个资产的到期时间上限
//
// 续期是「在原有效期上加」，不设上限的话 30 天档反复续费能把资产刷成事实上的永久，
// 也就绕过了「限定勋章」的设计（后台改 expire_days 不追溯，只能靠续期叠加）。
const renewMaxYears = 10

// renewMaxDays - 单个档位的最大天数（与后台表单上限一致）
const renewMaxDays = 3650

// RenewOption - 一个续期档位（多久 / 多少钱）
type RenewOption struct {
	Days int `json:"days"`
	// Price 售价（积分，0 = 免费续期）
	Price int `json:"price"`
	// OriginPrice 划线原价（0 = 不展示；不高于 Price 时不展示）
	OriginPrice int    `json:"origin_price"`
	Tag         string `json:"tag"`
}

// RenewConfig - 一个商品的续期配置
type RenewConfig struct {
	Enabled bool          `json:"enabled"`
	Options []RenewOption `json:"options"`
}

// renewRawMap - 把 renew 统一成 map[string]any
//
// utils.Json.Decode 内部走 cast.ToString，只认字符串：后台表单直接 POST 上来的是
// map[string]any，经过 AfterFind 的解码结果也是 map —— 这些都不能再过 Decode，
// 否则会被当成空配置，续期档位「配了却不生效」。
func renewRawMap(raw any) map[string]any {
	switch raw.(type) {
	case nil:
		return nil
	case string, []byte:
		return cast.ToStringMap(utils.Json.Decode(raw))
	default:
		return cast.ToStringMap(raw)
	}
}

// ParseRenewConfig - 把库里/表单里的 renew 反序列化并规整
//
// 宽松解析：脏数据（天数为 0、重复档位、价格倒挂）在这里就被丢掉，
// 不让它一路走到下单逻辑里；天数重复时保留第一条并按天数升序排列，
// 前台因此天然是「30 天 → 90 天 → 一年」的价格表顺序。
func ParseRenewConfig(raw any) RenewConfig {

	config := RenewConfig{}

	row := renewRawMap(raw)
	if len(row) == 0 {
		return config
	}

	config.Enabled = cast.ToBool(row["enabled"])

	seen := make(map[int]bool)
	for _, item := range cast.ToSlice(row["options"]) {
		opt := cast.ToStringMap(item)

		days := cast.ToInt(opt["days"])
		if days <= 0 || days > renewMaxDays || seen[days] {
			continue
		}

		price := cast.ToInt(opt["price"])
		if price < 0 {
			price = 0
		}

		origin := cast.ToInt(opt["origin_price"])
		// 划线价只在「高于售价」时有意义：填反了前台会显示「反而更贵」的假折扣
		if origin <= price {
			origin = 0
		}

		seen[days] = true
		config.Options = append(config.Options, RenewOption{
			Days:        days,
			Price:       price,
			OriginPrice: origin,
			Tag:         cast.ToString(opt["tag"]),
		})
	}

	sort.Slice(config.Options, func(i, j int) bool {
		return config.Options[i].Days < config.Options[j].Days
	})

	return config
}

// NormalizeRenewConfig - 后台保存前的严格校验，返回可直接落库的结构
//
// 与 ParseRenewConfig 的区别：这里不静默丢数据，配置错了直接报错让管理员改，
// 避免「后台填了三个档位，前台只显示两个」这种哑巴问题。
func NormalizeRenewConfig(raw any) (map[string]any, error) {

	if raw == nil || utils.Is.Empty(raw) {
		return map[string]any{"enabled": false, "options": []any{}}, nil
	}

	row := renewRawMap(raw)
	if len(row) == 0 {
		return nil, errors.New("续期配置格式不正确！")
	}

	enabled := cast.ToBool(row["enabled"])
	result := map[string]any{"enabled": enabled, "options": []any{}}

	seen := make(map[int]bool)
	for _, item := range cast.ToSlice(row["options"]) {
		opt := cast.ToStringMap(item)

		days := cast.ToInt(opt["days"])
		if days <= 0 || days > renewMaxDays {
			return nil, fmt.Errorf("续期天数需在 1 ~ %d 之间！", renewMaxDays)
		}
		if seen[days] {
			return nil, fmt.Errorf("续期档位「%d 天」重复了！", days)
		}

		price := cast.ToInt(opt["price"])
		if price < 0 {
			return nil, errors.New("续期价格不能为负数！")
		}

		origin := cast.ToInt(opt["origin_price"])
		if origin > 0 && origin < price {
			return nil, fmt.Errorf("「%d 天」档的划线原价不能低于售价！", days)
		}
		if origin <= price {
			origin = 0
		}

		seen[days] = true
		result["options"] = append(result["options"].([]any), map[string]any{
			"days":         days,
			"price":        price,
			"origin_price": origin,
			"tag":          cast.ToString(opt["tag"]),
		})
	}

	if enabled && len(result["options"].([]any)) == 0 {
		return nil, errors.New("开启续期后至少要配置一个档位！")
	}

	return result, nil
}

// option - 按天数取档位（下单时只认天数，价格由这里给出）
func (this RenewConfig) option(days int) (RenewOption, bool) {
	for _, item := range this.Options {
		if item.Days == days {
			return item, true
		}
	}
	return RenewOption{}, false
}

// Result - 前台展示用档位（补天数文案、折扣、是否免费）
func (this RenewConfig) Result() []facade.H {

	list := make([]facade.H, 0, len(this.Options))
	if !this.Enabled {
		return list
	}

	for _, item := range this.Options {
		discount := ""
		if item.OriginPrice > item.Price && item.OriginPrice > 0 {
			rate := float64(item.Price) / float64(item.OriginPrice) * 10
			if rate <= 0 {
				discount = "免费"
			} else {
				discount = fmt.Sprintf("%.1f 折", rate)
			}
		}

		list = append(list, facade.H{
			"days":         item.Days,
			"days_text":    RenewDurationText(item.Days),
			"price":        item.Price,
			"origin_price": item.OriginPrice,
			"tag":          item.Tag,
			"discount":     discount,
			"free":         item.Price <= 0,
		})
	}

	return list
}

// RenewOptionsResult - 前台取某件资产的续期档位（未配置 / 已关闭时返回空数组）
func RenewOptionsResult(raw any) []facade.H {
	return ParseRenewConfig(raw).Result()
}

// RenewDurationText - 天数的人类可读文案（30 → 30 天，180 → 6 个月，365 → 1 年）
func RenewDurationText(days int) string {
	switch {
	case days <= 0:
		return ""
	case days%365 == 0:
		return fmt.Sprintf("%d 年", days/365)
	case days%30 == 0 && days >= 60:
		return fmt.Sprintf("%d 个月", days/30)
	default:
		return fmt.Sprintf("%d 天", days)
	}
}

// RenewExtend - 计算续期后的到期时间
//
// 语义与淘宝 / 京东的「续费」一致：
//   - 还在有效期内：在原到期时间上叠加，用户已经买过的时间不浪费；
//   - 已经过期：从当前时间重新起算，不往回补（否则过期用户等于白拿）；
//   - 单次结果不超过 now + renewMaxYears 年。
func RenewExtend(currentExpire int64, days int) int64 {
	now := time.Now().Unix()

	base := currentExpire
	if base < now {
		base = now
	}

	next := base + int64(days)*86400
	if limit := now + int64(renewMaxYears)*365*86400; next > limit {
		next = limit
	}

	return next
}

// renewCharge - 事务内扣积分并写流水（price=0 时只写流水，不扣款）
//
// 与装扮商城一致：原子条件更新 `integral >= price`，余额不足时影响 0 行，
// 避免并发把余额扣成负数。
func renewCharge(tx *gorm.DB, uid int, option RenewOption, description string, extra map[string]any) (int, error) {

	var userRow Users
	if err := tx.Model(&Users{}).Where("id", uid).First(&userRow).Error; err != nil {
		return 0, errors.New("用户不存在！")
	}

	balance := userRow.Integral

	if option.Price > 0 {
		res := tx.Model(&Users{}).
			Where("id", uid).
			Where("integral >= ?", option.Price).
			UpdateColumn("integral", gorm.Expr("integral - ?", option.Price))
		if res.Error != nil {
			return 0, res.Error
		}
		if res.RowsAffected == 0 {
			return 0, errors.New("积分不足！")
		}
		balance -= option.Price
	}

	payload := map[string]any{
		"days":          option.Days,
		"price":         option.Price,
		"origin_price":  option.OriginPrice,
		"balance_after": balance,
	}
	for key, val := range extra {
		payload[key] = val
	}

	if err := tx.Create(&Integral{
		Uid:         uid,
		Value:       -option.Price,
		Type:        IntegralTypeRenew,
		Description: description,
		Json:        utils.Json.Encode(payload),
	}).Error; err != nil {
		return 0, err
	}

	return balance, nil
}

// renewTarget - 续期前先确认「这件商品配了续期」且「这个档位真实存在」
func renewTarget(raw any, days int) (RenewOption, error) {

	config := ParseRenewConfig(raw)
	if !config.Enabled || len(config.Options) == 0 {
		return RenewOption{}, errors.New("该商品暂不支持续期！")
	}

	option, ok := config.option(days)
	if !ok {
		return RenewOption{}, errors.New("续期档位不存在，请刷新后重试！")
	}

	return option, nil
}

// RenewBadge - 勋章续期（事务：校验 → 扣积分 → 延长到期时间 → 写流水）
//
//   - 只支持「已持有」的勋章，且必须是有限期的（永久勋章传进来直接拒绝）；
//   - 已过期的勋章也能续：记录还在，续期后重新变为可用；
//   - 续期后重置 ExpireNotified，新一轮有效期还会收到「即将到期」提醒。
func RenewBadge(uid, badgeId, days int) (result facade.H, err error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	badgeRow, _ := facade.DB.Model(&Badge{}).Where("id", badgeId).Find()
	if utils.Is.Empty(badgeRow) {
		return nil, errors.New("勋章不存在！")
	}

	badge := cast.ToStringMap(badgeRow)
	badgeName := cast.ToString(badge["name"])

	option, err := renewTarget(badge["renew"], days)
	if err != nil {
		return nil, err
	}

	record, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Find()
	if utils.Is.Empty(record) {
		return nil, errors.New("你还没有这枚勋章，无法续期！")
	}

	current := cast.ToInt64(cast.ToStringMap(record)["expire_time"])
	if current <= 0 {
		return nil, errors.New("这枚勋章是永久有效的，不需要续期！")
	}

	next := RenewExtend(current, option.Days)

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		balance, chargeErr := renewCharge(tx, uid, option, "续期勋章："+badgeName, map[string]any{
			"badge_id":      badgeId,
			"badge_name":    badgeName,
			"expire_before": current,
			"expire_after":  next,
		})
		if chargeErr != nil {
			return chargeErr
		}

		if err := tx.Model(&UserBadge{}).
			Where("uid", uid).
			Where("badge_id", badgeId).
			Updates(map[string]any{
				"expire_time":     next,
				"expire_notified": 0,
			}).Error; err != nil {
			return err
		}

		result = facade.H{
			"badge_id":      badgeId,
			"name":          badgeName,
			"days":          option.Days,
			"price":         option.Price,
			"integral":      balance,
			"expire_before": current,
			"expire_after":  next,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 我的勋章带缓存（BadgeProgress），续期后必须失效，否则页面还是「已过期」
	delBadgeCache(uid)

	return result, nil
}

// RenewDecoration - 装扮续期（事务：校验 → 扣积分 → 延长到期时间 → 写流水）
//
// 与装扮商城的 BuyDecoration 的区别：买是「获得 / 重置有效期」，续期是「在原有效期上加天数」。
// 已经过期（status=expired）的装扮续期后状态回到「已拥有」，可以重新佩戴。
func RenewDecoration(uid, decorationId, days int) (result facade.H, err error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	decoRow, _ := facade.DB.Model(&Decoration{}).Where("id", decorationId).Find()
	if utils.Is.Empty(decoRow) {
		return nil, errors.New("装扮不存在！")
	}

	deco := cast.ToStringMap(decoRow)
	name := cast.ToString(deco["name"])

	option, err := renewTarget(deco["renew"], days)
	if err != nil {
		return nil, err
	}

	record, _ := facade.DB.Model(&UserDecoration{}).Where("uid", uid).Where("decoration_id", decorationId).Find()
	if utils.Is.Empty(record) {
		return nil, errors.New("你还没有这件装扮，无法续期！")
	}

	row := cast.ToStringMap(record)
	current := cast.ToInt64(row["expire_time"])
	if current <= 0 {
		return nil, errors.New("这件装扮是永久有效的，不需要续期！")
	}

	next := RenewExtend(current, option.Days)

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		balance, chargeErr := renewCharge(tx, uid, option, "续期装扮："+name, map[string]any{
			"decoration_id":   decorationId,
			"decoration_name": name,
			"expire_before":   current,
			"expire_after":    next,
		})
		if chargeErr != nil {
			return chargeErr
		}

		updates := map[string]any{
			"expire_time":     next,
			"expire_notified": 0,
		}

		// 过期状态续期后回到「已拥有」：否则到期判断仍按 status 走，续了也戴不上
		if cast.ToInt(row["status"]) == UserDecorationStatusExpired {
			updates["status"] = UserDecorationStatusOwned
		}

		if err := tx.Model(&UserDecoration{}).
			Where("uid", uid).
			Where("decoration_id", decorationId).
			Updates(updates).Error; err != nil {
			return err
		}

		result = facade.H{
			"decoration_id": decorationId,
			"name":          name,
			"days":          option.Days,
			"price":         option.Price,
			"integral":      balance,
			"expire_before": current,
			"expire_after":  next,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
