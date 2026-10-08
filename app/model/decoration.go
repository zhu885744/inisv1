package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// jsonMapOf - 兼容「已解码的 map」与「原始 JSON 字符串」两种形态
//
// Select() 读出的字段是 JSON 字符串，First(&struct) 经 AfterFind 解码后是 map，
// 两种都要能处理，否则会出现「数据明明存在却读不到」的问题。
func jsonMapOf(value any) map[string]any {
	if data := asStringMap(value); data != nil {
		return data
	}
	text := cast.ToString(value)
	if utils.Is.Empty(text) {
		return nil
	}
	return asStringMap(utils.Json.Decode(text))
}

// ============================== 类型元数据 ==============================

// DecorationTypeMeta - 装扮类型元数据
//
// 前端据此动态渲染「商城分区」与「装扮列表」，新增装扮类型时只需在这里补一项，
// 前端不需要为新类型写任何判断分支（payload.fields 描述了该类型支持哪些样式字段）。
func DecorationTypeMeta() []facade.H {
	return []facade.H{
		{
			"type":              DecorationTypeAvatarFrame,
			"name":              "头像框",
			"icon":              "bi-person-badge",
			"slot":              "avatar",
			"allow_custom_text": 0,
			"fields":            []string{"image", "scale", "animated"},
			"description":       "展示在头像外圈，可随时切换。",
		},
		{
			"type":              DecorationTypeTitle,
			"name":              "头衔",
			"icon":              "bi-award",
			"slot":              "title",
			"allow_custom_text": 1,
			"fields":            []string{"text", "color", "bg", "border", "glow", "icon", "animated"},
			"description":       "展示在昵称旁，支持配色、渐变与发光。",
		},
	}
}

// DecorationTypeName - 装扮类型显示名
func DecorationTypeName(typ string) string {
	for _, item := range DecorationTypeMeta() {
		if cast.ToString(item["type"]) == typ {
			return cast.ToString(item["name"])
		}
	}
	return "装扮"
}

// DecorationTypeSupported - 是否为受支持的装扮类型
//
// 只认 DecorationTypeMeta 里声明过的类型，避免脏数据让前端渲染出错。
func DecorationTypeSupported(typ string) bool {
	for _, item := range DecorationTypeMeta() {
		if cast.ToString(item["type"]) == typ {
			return true
		}
	}
	return false
}

// ============================== 配置 ==============================

// defaultDecorationConfig - 默认装扮配置
func defaultDecorationConfig() facade.H {
	return facade.H{
		"enabled":       1,
		"show_in_goods": 1,
		"currency":      DecorationPriceIntegral,
		"avatar_frame": facade.H{
			"enabled":     1,
			"frame_scale": 1.2,
		},
		"title": facade.H{
			"enabled":      1,
			"allow_custom": 1,
			"max_length":   10,
			"presets":      defaultTitles(),
		},
	}
}

// mergeDecorationConfig - 用后台配置覆盖默认配置（含一级子对象合并）
func mergeDecorationConfig(base facade.H, override map[string]any) facade.H {
	for key, value := range override {
		if value == nil {
			continue
		}
		child := asStringMap(value)
		if child != nil {
			current := asStringMap(base[key])
			if current == nil {
				current = map[string]any{}
			}
			for childKey, childValue := range child {
				current[childKey] = childValue
			}
			base[key] = current
			continue
		}
		base[key] = value
	}
	return base
}

// GetDecorationConfig - 获取装扮配置（缓存优先，缺失项用默认值兜底）
func GetDecorationConfig() facade.H {
	config := defaultDecorationConfig()

	if facade.Cache.Has(DecorationConfigKey) {
		if data, ok := facade.Cache.Get(DecorationConfigKey).(map[string]any); ok {
			return mergeDecorationConfig(config, data)
		}
		if data, ok := facade.Cache.Get(DecorationConfigKey).(facade.H); ok {
			return mergeDecorationConfig(config, data)
		}
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", DecorationConfigKey).Find()
	if !utils.Is.Empty(item) {
		if jsonData, ok := item["json"].(map[string]any); ok {
			config = mergeDecorationConfig(config, jsonData)
			facade.Cache.Set(DecorationConfigKey, config)
		}
	}

	return config
}

// DecorationEnabled - 装扮功能总开关
func DecorationEnabled() bool {
	return cast.ToInt(GetDecorationConfig()["enabled"]) == 1
}

// ============================== 列表 ==============================

// ShopDecorations - 装扮商城列表（仅上架，价格 / 库存 / 限购全部取自装扮自身）
//
// 装扮商城是独立分区：不再依赖积分商城的商品记录（Goods）。
// 兑换走 BuyDecoration，只与「用户积分」联动。
// uid > 0 时会带上该用户的限购剩余（-1 = 不限购）。
//
// typ       类型筛选（avatar_frame / title…，空=全部）
// priceType 价格筛选（free 免费 / paid 付费 / 空=全部，见 FilterDecorationPrice）
func ShopDecorations(typ string, priceType string, uid int) []facade.H {

	// 注意：facade 的 Select() 要求 dest 是**切片**。
	// 传单结构体（&Decoration{}）时框架内部会把结果编码成 JSON 对象，
	// cast.ToSlice 解析对象恒为长度 0，表现为「商城有数据却永远查不到」。
	var decoList []Decoration
	query := facade.DB.Model(&decoList).Where("status", DecorationStatusOn)
	if !utils.Is.Empty(typ) {
		query = query.Where("type", typ)
	}
	query = FilterDecorationPrice(query, priceType)
	list, _ := query.Order("sort desc, id asc").Select()

	// 该用户已兑换过的装扮数量（限购剩余用），一次查询避免 N+1
	boughtMap := map[int]int{}
	if uid > 0 {
		var ownList []UserDecoration
		owns, _ := facade.DB.Model(&ownList).Where("uid", uid).Select()
		for _, item := range owns {
			boughtMap[cast.ToInt(item["decoration_id"])]++
		}
	}

	result := make([]facade.H, 0, len(list))
	for _, item := range list {
		id := cast.ToInt(item["id"])
		isDefault := cast.ToInt(item["is_default"]) == 1
		price := cast.ToInt(item["price"])
		stock := cast.ToInt(item["stock"])
		limit := cast.ToInt(item["limit_per_user"])

		item["payload"] = decorationPayloadMap(item["payload"])
		item["is_default"] = isDefault
		item["price"] = price
		item["stock"] = stock
		item["unlimited"] = stock < 0
		item["sold_out"] = stock == DecorationStockSoldOut

		// 有效期（秒，0=永久）：前台商城卡片直接展示，避免用户兑换后才发现有期限
		duration := cast.ToInt64(item["duration"])
		item["duration"] = duration
		item["permanent"] = duration <= 0

		// 能否兑换：默认装扮 / 未定价 / 售罄 → 不可兑换，并给出原因
		canBuy, reason := true, ""
		switch {
		case isDefault:
			canBuy, reason = false, "系统默认装扮"
		case price <= 0:
			// 免费装扮（活动赠品等）只能在装扮管理里发放，商城不出售
			canBuy, reason = false, "暂未上架"
		case stock == DecorationStockSoldOut:
			canBuy, reason = false, "已售罄"
		}

		// 限购剩余（-1 表示不限购）
		remain := -1
		if limit > 0 {
			remain = limit - boughtMap[id]
			if remain <= 0 {
				remain = 0
				if canBuy {
					canBuy, reason = false, "已达限购"
				}
			}
		}
		item["limit_remain"] = remain
		item["on_sale"] = canBuy
		item["buy_reason"] = reason

		result = append(result, item)
	}

	return result
}

// BuyDecoration - 兑换装扮（事务：校验 → 扣库存 → 扣积分 → 写流水 → 发放）
//
// 与积分商城完全解耦：不读也不写 goods / goods_order，价格、库存、限购、经验门槛
// 全部取自装扮自身。兑换痕迹落在两处，便于用户与后台回溯：
//   - 积分流水（Integral.Type = decoration，含余额快照）
//   - 用户装扮表（UserDecoration：来源 shop / 获得时间 / 过期时间）
//
// 任一步失败整体回滚，积分不会白扣。
func BuyDecoration(uid int, decorationId int) (result facade.H, err error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}
	if cast.ToInt(GetDecorationConfig()["enabled"]) != 1 {
		return nil, errors.New("装扮商城未开启！")
	}

	// 事务外通知管理员用（昵称在事务内读取）
	nickname, decoName, price := "", "", 0

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 1. 装扮校验
		var deco Decoration
		if err := tx.Where("id = ?", decorationId).First(&deco).Error; err != nil {
			return errors.New("装扮不存在！")
		}
		if deco.Status != DecorationStatusOn {
			return errors.New("装扮已下架！")
		}
		if !DecorationTypeSupported(deco.Type) {
			return errors.New("暂不支持的装扮类型！")
		}
		if deco.IsDefault == 1 {
			return errors.New("系统默认装扮无需兑换！")
		}
		if deco.Price <= 0 {
			return errors.New("该装扮当前不可兑换！")
		}
		if deco.Stock == DecorationStockSoldOut {
			return errors.New("该装扮已售罄！")
		}

		// 2. 用户校验（经验门槛 / 每人限购）
		var user Users
		if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
			return errors.New("用户不存在！")
		}
		if deco.MinExp > 0 && user.Exp < deco.MinExp {
			return fmt.Errorf("经验值达到 %d 才可兑换！", deco.MinExp)
		}
		if deco.LimitPerUser > 0 {
			var bought int64
			if err := tx.Model(&UserDecoration{}).
				Where("uid", uid).
				Where("decoration_id", decorationId).
				Count(&bought).Error; err != nil {
				return err
			}
			if bought >= int64(deco.LimitPerUser) {
				return errors.New("已达每人限购上限！")
			}
		}

		// 3. 扣库存（原子条件更新，防超卖；-1 表示不限量）
		//
		// 注意：这里必须写占位符形式 Where("stock > ?", 0)。
		// 三参形式 Where("字段", ">", 值) 只有项目自封装的 facade.DB.Model(...) 支持，
		// 原生 *gorm.DB 会把它当「主键列表」处理 —— 生成 `id IN ('stock','>',0)`，
		// MySQL 转换非数字字符串时报 1292 (Truncated incorrect DOUBLE value)。
		if deco.Stock > 0 {
			res := tx.Model(&Decoration{}).
				Where("id", decorationId).
				Where("stock > ?", 0).
				UpdateColumn("stock", gorm.Expr("stock - 1"))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("该装扮已售罄！")
			}
		}

		// 4. 扣积分（原子条件更新：余额不足时影响 0 行，避免并发扣成负数）
		//
		// 同样必须用占位符：写成 Where("integral", ">=", deco.Price) 时原生 gorm
		// 会生成 `WHERE id = ? AND id IN ('integral','>=',1000)`，
		// 于是 MySQL 在把 'integral' 转成数字时报：
		// Error 1292 (22007): Truncated incorrect DOUBLE value: 'integral'
		res := tx.Model(&Users{}).
			Where("id", uid).
			Where("integral >= ?", deco.Price).
			UpdateColumn("integral", gorm.Expr("integral - ?", deco.Price))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("积分不足！")
		}

		// 5. 装扮获得量 +1
		if err := tx.Model(&Decoration{}).Where("id = ?", decorationId).
			UpdateColumn("sold", gorm.Expr("sold + 1")).Error; err != nil {
			return err
		}

		balance := user.Integral - deco.Price
		nickname, decoName, price = user.Nickname, deco.Name, deco.Price

		// 6. 积分流水（装扮消费单独一种类型，便于后台区分统计）
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       -deco.Price,
			Type:        IntegralTypeDecoration,
			Description: "兑换装扮：" + deco.Name,
			Json: utils.Json.Encode(map[string]any{
				"decoration_id":   deco.Id,
				"decoration_name": deco.Name,
				"balance_after":   balance,
			}),
		}).Error; err != nil {
			return err
		}

		// 7. 发放装扮（同事务：失败则整体回滚，积分一并退回）
		record, err := GrantDecorationTx(tx, uid, decorationId, DecorationSourceShop, 0, 0)
		if err != nil {
			return err
		}

		result = facade.H{
			"decoration_id": deco.Id,
			"name":          deco.Name,
			"type":          deco.Type,
			"price":         deco.Price,
			"integral":      balance,
			"expire_time":   record.ExpireTime,
			"duration":      deco.Duration,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 管理员站内信：装扮销售提醒（无开关，落库即达）。
	// 放在事务提交后：只有真正兑换成功才提醒，回滚的不打扰运营。
	// 用户在事务内已收到「获得装扮」通知（见 GrantDecorationTx），这里不重复发给用户。
	NotifyAdmins(uid, NotificationTypeDecoration, "有用户兑换了装扮",
		fmt.Sprintf("用户：%s（ID %d） · 装扮：%s · 消耗积分：%d", nickname, uid, decoName, price),
		"decoration", decorationId)

	return result, nil
}

// MyDecorations - 我的装扮（已拥有 + 系统默认），按类型分组
//
// 返回结构：
//
//	{ types: [...类型元数据], wearing: {type: 装扮}, list: {type: [装扮...]} }
func MyDecorations(uid int) facade.H {

	result := facade.H{
		"types":   DecorationTypeMeta(),
		"wearing": facade.H{},
		"list":    facade.H{},
	}

	grouped := facade.H{}
	for _, meta := range DecorationTypeMeta() {
		grouped[cast.ToString(meta["type"])] = []facade.H{}
	}
	result["list"] = grouped

	if uid <= 0 {
		return result
	}

	var user Users
	if err := facade.DB.Drive().Where("id = ?", uid).First(&user).Error; err != nil {
		return result
	}

	wearing := wearingMap(user.Json)
	result["wearing"] = WearingDecorations(uid)

	// 我拥有的装扮（含已过期，前端需要提示「已过期」）
	// Select() 的 dest 必须是切片，否则恒为空（见 ShopDecorations 注释）
	var ownList []UserDecoration
	owns, _ := facade.DB.Model(&ownList).Where("uid", uid).Select()
	ownMap := map[int]facade.H{}
	ids := make([]int, 0, len(owns))
	for _, item := range owns {
		id := cast.ToInt(item["decoration_id"])
		if id <= 0 {
			continue
		}
		ownMap[id] = item
		ids = append(ids, id)
	}

	// 系统默认装扮：所有用户默认拥有
	var defaultList []Decoration
	defaults, _ := facade.DB.Model(&defaultList).
		Where("status", DecorationStatusOn).
		Where("is_default", 1).Select()
	for _, item := range defaults {
		id := cast.ToInt(item["id"])
		if _, exist := ownMap[id]; !exist {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return result
	}

	var decoList []Decoration
	list, _ := facade.DB.Model(&decoList).Where("id", "in", ids).
		Order("sort desc, id asc").Select()

	for _, item := range list {
		id := cast.ToInt(item["id"])
		typ := cast.ToString(item["type"])
		isDefault := cast.ToInt(item["is_default"]) == 1

		own := ownMap[id]
		expireTime := int64(0)
		customText := ""
		source := DecorationSourceDefault
		if own != nil {
			expireTime = cast.ToInt64(own["expire_time"])
			customText = cast.ToString(own["custom_text"])
			source = cast.ToString(own["source"])
		}

		item["payload"] = decorationPayloadMap(item["payload"])
		item["is_default"] = isDefault
		item["owned"] = true
		item["expire_time"] = expireTime
		item["expired"] = !isDefault && UserDecorationExpired(expireTime)
		// 装扮被后台下架后仍要展示（用户确实拥有），但不能再佩戴（WearDecoration 会拒绝）。
		// 前端据此置灰并说明原因，避免用户点了才收到「装扮已下架」的报错。
		item["off_shelf"] = cast.ToInt(item["status"]) != DecorationStatusOn
		item["custom_text"] = customText
		item["source"] = source
		item["wearing"] = cast.ToInt(wearing[typ]) == id

		list, ok := grouped[typ].([]facade.H)
		if !ok {
			list = []facade.H{}
		}
		grouped[typ] = append(list, item)
	}

	result["list"] = grouped
	return result
}

// DecorationCount - 装扮统计（后台概览用）
func DecorationCount() facade.H {
	total, _ := facade.DB.Model(&Decoration{}).Count()
	onSale, _ := facade.DB.Model(&Decoration{}).Where("status", DecorationStatusOn).Count()
	owned, _ := facade.DB.Model(&UserDecoration{}).Count()

	byType := facade.H{}
	for _, meta := range DecorationTypeMeta() {
		typ := cast.ToString(meta["type"])
		count, _ := facade.DB.Model(&Decoration{}).Where("type", typ).Count()
		byType[typ] = count
	}

	return facade.H{
		"total":   total,
		"on_sale": onSale,
		"owned":   owned,
		"by_type": byType,
	}
}


// DecorationConfigKey - 装扮配置缓存键（与 Config 表的 key 一致）
const DecorationConfigKey = "SYSTEM_DECORATION"

// 装扮类型常量
//
// 这是整个装扮体系唯一需要「扩展」的地方：新增一类装扮 = 新增一个 type 常量 +
// 一个 payload 渲染约定 + 前端一个渲染器，不需要改动表结构。
// 已按 BBS 方向预留：勋章 / 昵称色 / 名片背景 / 评论气泡 / 主题。
const (
	DecorationTypeAvatarFrame = "avatar_frame" // 头像框
	DecorationTypeTitle       = "title"        // 头衔（称号）
	DecorationTypeMedal       = "medal"        // 勋章（预留）
	DecorationTypeNameColor   = "name_color"   // 昵称颜色（预留）
	DecorationTypeCardBg      = "card_bg"      // 名片背景（预留）
	DecorationTypeBubble      = "bubble"       // 评论气泡（预留）
)

// 装扮上下架状态
const (
	DecorationStatusOff = 0 // 下架
	DecorationStatusOn  = 1 // 上架
)

// 装扮库存（独立于积分商城，取值语义与商品的 stock 不同）
const (
	DecorationStockUnlimited = -1 // 不限量（默认）
	DecorationStockSoldOut   = 0  // 已售罄
)

// 装扮价格类型（用于展示与生成商城商品；实际扣减以商品记录为准）
const (
	DecorationPriceIntegral = "integral" // 积分
	DecorationPriceExp      = "exp"      // 经验
	DecorationPriceFree     = "free"     // 免费
)

// DecorationPricePaid - 「付费」筛选用的伪类型：非免费（积分 / 经验）即付费。
// 前台与后台只暴露「全部 / 免费 / 付费」三档，具体价格类型仍由 price_type 精确匹配兜底。
const DecorationPricePaid = "paid"

// FilterDecorationPrice - 装扮「免费 / 付费」筛选（商城列表与后台列表共用同一口径，避免两处判断不一致）
//
//	"" / all → 不过滤
//	free     → 免费（price_type = free）
//	paid     → 付费（price_type != free，含 integral / exp）
//	其他值   → 按 price_type 精确匹配（integral / exp）
func FilterDecorationPrice(query *facade.ModelStruct, value string) *facade.ModelStruct {
	switch value {
	case "", "all":
		return query
	case DecorationPriceFree:
		return query.Where("price_type", DecorationPriceFree)
	case DecorationPricePaid:
		return query.Where("price_type", "!=", DecorationPriceFree)
	}
	return query.Where("price_type", value)
}

// 用户装扮状态
const (
	UserDecorationStatusOwned   = 0 // 已拥有（未佩戴）
	UserDecorationStatusWearing = 1 // 佩戴中
	UserDecorationStatusExpired = 2 // 已过期
)

// 装扮来源（记录在 UserDecoration.source，便于多渠道发放与统计）
const (
	DecorationSourceShop     = "shop"     // 商城购买
	DecorationSourceAdmin    = "admin"    // 管理员发放
	DecorationSourceActivity = "activity" // 活动赠送
	DecorationSourceLevel    = "level"    // 等级解锁
	DecorationSourceGacha    = "gacha"    // 抽卡获得（预留）
	DecorationSourceDefault  = "default"  // 系统默认
)

// Decoration - 装扮定义表
//
// 只描述「长什么样」，不描述「怎么获得」：获得渠道由商城商品（Goods.DecorationId）
// 或后台直接发放承担，因此同一个装扮既能上架售卖，也能作为活动奖励 / 等级解锁 / 默认装扮。
type Decoration struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Type        string `gorm:"size:32; index; comment:装扮类型（avatar_frame/title/...）; default:'avatar_frame';" json:"type"`
	Name        string `gorm:"size:128; comment:装扮名称;" json:"name"`
	Description string `gorm:"type:text; comment:装扮描述; default:Null;" json:"description"`
	Preview     string `gorm:"comment:预览图; default:Null;" json:"preview"`
	// Payload 样式数据（JSON）：类型不同结构不同，前端按 type 渲染
	//
	//	title        {"text":"掌门","color":"#fff","bg":"linear-gradient(...)","glow":"...","icon":"bi-award"}
	//	avatar_frame {"image":"https://.../1.gif","scale":1.2,"animated":true}
	Payload any `gorm:"type:longtext; comment:样式数据（JSON）; default:Null;" json:"payload"`
	// 价格（仅作展示 / 生成商城商品时的初值，真正扣减看关联的商品记录）
	PriceType string `gorm:"size:16; comment:价格类型（integral/exp/free）; default:'integral';" json:"price_type"`
	Price     int    `gorm:"type:int(32); comment:价格; default:0;" json:"price"`
	// Rarity 稀有度（BBS 抽卡 / 掉落用）
	Rarity string `gorm:"size:16; comment:稀有度（normal/rare/epic/legend）; default:'normal';" json:"rarity"`
	// Duration 有效期（秒，0=永久）；发放后写入 UserDecoration.ExpireTime
	Duration int64 `gorm:"comment:有效期（秒，0=永久）; default:0;" json:"duration"`
	// Unlock 解锁条件（JSON：{"level":3,"exp":100}），P0 仅存储与展示，后续接入校验
	Unlock   any    `gorm:"type:longtext; comment:解锁条件（JSON）; default:Null;" json:"unlock"`
	Category string `gorm:"size:32; index; comment:分组（自由字符串）; default:Null;" json:"category"`
	Status   int    `gorm:"tinyint; index; default:1; comment:状态（0下架 1上架）;" json:"status"`
	Sort     int    `gorm:"type:int(32); index; comment:排序权重（越大越靠前）; default:0;" json:"sort"`
	Sold     int    `gorm:"type:int(32); comment:已获得数量; default:0;" json:"sold"`
	// 以下为「装扮商城」自身的售卖字段（不依赖积分商城的商品记录）
	// Stock 库存：-1=不限量（默认）0=已售罄 >0=剩余数量
	Stock int `gorm:"type:int(32); comment:库存（-1不限 0售罄 >0剩余）; default:-1;" json:"stock"`
	// LimitPerUser 每人限购数量（0=不限购）
	LimitPerUser int `gorm:"type:int(32); comment:每人限购数量（0=不限购）; default:0;" json:"limit_per_user"`
	// MinExp 兑换所需最低经验值（0=不限）
	MinExp int `gorm:"type:int(32); comment:兑换所需最低经验值（0=不限）; default:0;" json:"min_exp"`
	// IsDefault 系统默认装扮：所有用户默认拥有（无需购买，也不写入用户装扮表）
	IsDefault int `gorm:"tinyint; index; comment:系统默认（1=所有用户默认拥有）; default:0;" json:"is_default"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// UserDecoration - 用户装扮表（谁拥有什么、是否佩戴中、何时过期）
//
// 佩戴状态的「唯一真相」是 users.json.decorations（{type: decoration_id}），
// 本表的 status=wearing 由佩戴/卸下流程同步维护，便于后台统计与用户端列表展示。
type UserDecoration struct {
	Id           int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid          int    `gorm:"type:int(32); index; comment:用户ID;" json:"uid"`
	DecorationId int    `gorm:"type:int(32); index; comment:装扮ID;" json:"decoration_id"`
	Type         string `gorm:"size:32; index; comment:装扮类型（冗余，便于按类型查询）; default:'';" json:"type"`
	Status       int    `gorm:"tinyint; index; comment:状态（0已拥有 1佩戴中 2已过期）; default:0;" json:"status"`
	ExpireTime   int64  `gorm:"index; comment:过期时间（0=永久）; default:0;" json:"expire_time"`
	// ExpireNotified 是否已发过「即将过期」站内提醒（定时任务用；续期时重置为 0，保证新一轮仍会提醒）
	ExpireNotified int `gorm:"tinyint; comment:是否已发即将过期提醒（0否 1是）; default:0;" json:"expire_notified"`
	Source       string `gorm:"size:32; comment:获得来源（shop/admin/activity/level/gacha）; default:'shop';" json:"source"`
	OrderId      int    `gorm:"type:int(32); comment:关联订单ID; default:0;" json:"order_id"`
	WearTime     int64  `gorm:"comment:最近佩戴时间; default:0;" json:"wear_time"`
	// Text 用户为该装扮填写的自定义文本（如纯样式类头衔的自定义称号文字）
	Text string `gorm:"size:64; comment:自定义文本（可选）; default:Null;" json:"custom_text"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// ============================== 初始化 ==============================

// InitDecoration - 初始化装扮表 + 首次运行导入默认装扮
//
// 注意：必须在 InitGoods 之后注册（见 base.go 的 InitTable 顺序），
// 因为默认装扮会同时生成一条关联的商品记录，让商城开箱即用。
func InitDecoration() {
	if err := facade.DB.Drive().AutoMigrate(&Decoration{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Decoration表迁移失败")
		return
	}
	if err := facade.DB.Drive().AutoMigrate(&UserDecoration{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserDecoration表迁移失败")
		return
	}
	seedDefaultDecorations()
}

// defaultAvatarFrames - 默认头像框资源（与旧前端 Profile.vue 的 PRESET_FRAMES 一致，
// 迁移到数据库后前端不再硬编码）
func defaultAvatarFrames() []string {
	list := make([]string, 0, 20)
	for index := 1; index <= 20; index++ {
		ext := "png"
		if index%2 == 1 {
			ext = "gif"
		}
		list = append(list, fmt.Sprintf("https://img.zhuxu.asia/txk/%d.%s", index, ext))
	}
	return list
}

// defaultTitles - 默认头衔（旧前端 PRESET_TITLES 的迁移）
func defaultTitles() []string {
	return []string{"掌门", "长老", "护法", "内门弟子", "外门弟子", "炼气修士", "筑基修士", "结丹修士", "元婴老祖", "化神大能"}
}

// defaultTitleColors - 头衔配色（下标与 defaultTitles 对应）
func defaultTitleColors() []string {
	return []string{"#b7893f", "#8f6ad6", "#3f8fd6", "#3fa06a", "#6b7280", "#8a8a82", "#4f8f8a", "#7a6ad6", "#c04a6a", "#3f6ad6"}
}

// seedDefaultDecorations - 首次运行时导入默认装扮（仅在装扮表为空时执行）
//
// 头像框 / 头衔各只保留 1 个「系统默认」免费项（头像框 1、掌门），
// 其余全部为积分兑换的装扮（价格见下方各自定价规则）。
//
// 注意：只在装扮表为空时导入 —— 已经跑起来的站点不会因为改动这里而变化，
// 存量数据请在后台「装扮管理」里调整（取消默认标记 + 价格类型选积分 + 填价格）。
func seedDefaultDecorations() {
	count, _ := facade.DB.Model(&Decoration{}).Count()
	if count > 0 {
		return
	}

	now := time.Now().Unix()
	list := make([]Decoration, 0, 30)

	for index, url := range defaultAvatarFrames() {
		// 仅第 1 个头像框免费（系统默认），其余全部按积分定价
		free := index == 0
		item := Decoration{
			Type:        DecorationTypeAvatarFrame,
			Name:        fmt.Sprintf("头像框 %d", index+1),
			Description: "佩戴后立即生效，可在「我的装扮」随时切换。",
			Preview:     url,
			Payload:     utils.Json.Encode(facade.H{"image": url, "scale": 1.15, "animated": index%2 == 0}),
			PriceType:   DecorationPriceIntegral,
			Price:       60 + index*20,
			Rarity:      "normal",
			Status:      DecorationStatusOn,
			Sort:        100 - index,
		}
		if free {
			item.PriceType = DecorationPriceFree
			item.Price = 0
			item.IsDefault = 1
			item.Category = "默认头像框"
		}
		list = append(list, item)
	}

	titles := defaultTitles()
	colors := defaultTitleColors()
	for index, text := range titles {
		// 同样只保留第 1 个头衔（掌门）免费，其余全部按积分定价
		free := index == 0
		color := "#8a8a82"
		if index < len(colors) {
			color = colors[index]
		}
		item := Decoration{
			Type:        DecorationTypeTitle,
			Name:        text,
			Description: "佩戴后展示在你的昵称旁。",
			Payload: utils.Json.Encode(facade.H{
				"text":  text,
				"color": "#ffffff",
				"bg":    color,
				"glow":  color + "55",
			}),
			PriceType: DecorationPriceIntegral,
			Price:     100 + index*50,
			Rarity:    "normal",
			Status:    DecorationStatusOn,
			Sort:      100 - index,
		}
		if free {
			item.PriceType = DecorationPriceFree
			item.Price = 0
			item.IsDefault = 1
			item.Category = "默认头衔"
		}
		list = append(list, item)
	}

	for index := range list {
		list[index].CreateTime = now
		list[index].UpdateTime = now
	}

	if err := facade.DB.Drive().Create(&list).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "默认装扮写入失败")
		return
	}

	// 注：装扮商城已与积分商城解耦，不再为装扮生成商品记录。
	// 价格 / 库存 / 限购都在装扮自身维护，兑换走 decoration/buy（只扣积分）。
	facade.Log.Info(map[string]any{"decoration": len(list)}, "默认装扮初始化完成")
}

// decorationCategoryName - 装扮类型 → 商城分类名
func decorationCategoryName(typ string) string {
	switch typ {
	case DecorationTypeAvatarFrame:
		return "头像框"
	case DecorationTypeTitle:
		return "头衔"
	case DecorationTypeMedal:
		return "勋章"
	case DecorationTypeNameColor:
		return "昵称颜色"
	case DecorationTypeCardBg:
		return "名片背景"
	case DecorationTypeBubble:
		return "评论气泡"
	}
	return "装扮"
}

// ============================== Hook ==============================

// AfterFind - 查询Hook
func (this *Decoration) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	this.Payload = utils.Json.Decode(this.Payload)
	this.Unlock = utils.Json.Decode(this.Unlock)
	this.Result = this.result()
	return
}

// AfterFind - 查询Hook
func (this *UserDecoration) AfterFind(tx *gorm.DB) (err error) {
	this.Json = utils.Json.Decode(this.Json)
	this.Result = this.result()
	return
}

// result - 装扮派生信息（前端展示用）
func (this *Decoration) result() facade.H {
	name := this.Category
	if utils.Is.Empty(name) {
		name = decorationCategoryName(this.Type)
	}
	return facade.H{
		"type_name":     DecorationTypeName(this.Type),
		"is_free":       this.PriceType == DecorationPriceFree || this.Price <= 0,
		"is_default":    this.IsDefault == 1,
		"permanent":     this.Duration <= 0,
		"category_name": name,
	}
}

// result - 用户装扮派生信息
func (this *UserDecoration) result() facade.H {
	expired := UserDecorationExpired(this.ExpireTime)
	return facade.H{
		"expired":   expired,
		"wearing":   this.Status == UserDecorationStatusWearing && !expired,
		"permanent": this.ExpireTime <= 0,
	}
}

// UserDecorationExpired - 装扮是否已过期（0=永久有效）
func UserDecorationExpired(expireTime int64) bool {
	return expireTime > 0 && time.Now().Unix() > expireTime
}

// ============================== 发放 / 回收 ==============================

// decorationPayloadMap - 取装扮的样式数据（兼容 AfterFind 解码后的 map 与 Select 读出的 JSON 字符串）
func decorationPayloadMap(value any) map[string]any {
	return jsonMapOf(value)
}

// GrantDecorationTx - 发放装扮（事务内）
//
// 幂等：同一用户同一装扮只保留一条记录，重复发放视为续期（不会重复插入、不会覆盖佩戴状态）。
// duration <= 0 时使用装扮自身的有效期；两者都为 0 表示永久。
func GrantDecorationTx(tx *gorm.DB, uid int, decorationId int, source string, duration int64, orderId int) (record UserDecoration, err error) {

	if uid <= 0 {
		return record, errors.New("请先登录！")
	}
	if tx == nil {
		tx = facade.DB.Drive()
	}

	var deco Decoration
	if err = tx.Where("id = ?", decorationId).First(&deco).Error; err != nil {
		return record, errors.New("装扮不存在！")
	}

	now := time.Now().Unix()
	switch {
	case duration < 0:
		// 显式指定「永久」：忽略装扮自身的有效期
		duration = 0
	case duration == 0:
		// 未指定：沿用装扮自身的有效期
		duration = deco.Duration
	}
	expireTime := int64(0)
	if duration > 0 {
		expireTime = now + duration
	}
	if utils.Is.Empty(source) {
		source = DecorationSourceShop
	}

	// 已存在记录（含已回收的软删除记录）：续期并恢复，避免重复插入
	var exist UserDecoration
	found := tx.Unscoped().
		Where("uid", uid).
		Where("decoration_id", decorationId).
		First(&exist).Error == nil

	if found {
		status := exist.Status
		if status == UserDecorationStatusExpired {
			status = UserDecorationStatusOwned
		}
		updates := map[string]any{
			"type":        deco.Type,
			"status":      status,
			"expire_time": expireTime,
			"source":      source,
			"order_id":    orderId,
			"delete_time": 0,
			// 续期后重置提醒标记：新一轮有效期到期前仍会收到「即将过期」提醒
			"expire_notified": 0,
		}
		if err = tx.Unscoped().Model(&UserDecoration{}).Where("id = ?", exist.Id).Updates(updates).Error; err != nil {
			return record, err
		}
		exist.Type = deco.Type
		exist.Status = status
		exist.ExpireTime = expireTime
		exist.Source = source
		exist.OrderId = orderId
		record = exist
	} else {
		record = UserDecoration{
			Uid:          uid,
			DecorationId: decorationId,
			Type:         deco.Type,
			Status:       UserDecorationStatusOwned,
			ExpireTime:   expireTime,
			Source:       source,
			OrderId:      orderId,
		}
		if err = tx.Create(&record).Error; err != nil {
			return record, err
		}
	}

	// 装扮获得量 +1（用于后台统计与热度排序）
	_ = tx.Model(&Decoration{}).Where("id = ?", decorationId).
		UpdateColumn("sold", gorm.Expr("sold + 1")).Error

	// 站内通知：获得 / 续期统一在这里发，商城兑换、管理员发放、活动、等级解锁都能覆盖。
	// 与发放写在同一个事务里（装扮回滚时通知一并回滚），失败只记日志、不影响发放。
	title := "你获得了装扮：" + deco.Name
	if found {
		title = "装扮续期成功：" + deco.Name
	}
	if err := tx.Create(&Notification{
		Uid:      uid,
		Type:     NotificationTypeDecoration,
		Title:    title,
		Content:  decorationNotifyBody(deco, source, expireTime),
		BindId:   decorationId,
		BindType: "decoration",
	}).Error; err != nil {
		facade.Log.Warn(map[string]any{"uid": uid, "decoration_id": decorationId, "error": err.Error()}, "装扮获得通知发送失败")
	}

	return record, nil
}

// GrantDecoration - 发放装扮（事务外，供后台 / 活动 / 等级解锁调用）
func GrantDecoration(uid int, decorationId int, source string, duration int64) error {
	_, err := GrantDecorationTx(nil, uid, decorationId, source, duration, 0)
	return err
}

// RevokeDecoration - 回收装扮（后台使用）：删除拥有记录，若正在佩戴则一并卸下
func RevokeDecoration(uid int, decorationId int) error {

	if uid <= 0 || decorationId <= 0 {
		return errors.New("参数错误！")
	}

	var deco Decoration
	if err := facade.DB.Drive().Where("id = ?", decorationId).First(&deco).Error; err != nil {
		return errors.New("装扮不存在！")
	}

	// 佩戴中：先卸下（同时清理 users 上的渲染字段）
	wearing := IsWearingDecoration(uid, decorationId)
	if wearing {
		if err := UnwearDecoration(uid, deco.Type); err != nil {
			return err
		}
	}

	// 软删除拥有记录（Unscoped 硬删除会破坏幂等续期逻辑，这里保持软删除）
	if err := facade.DB.Drive().
		Where("uid", uid).
		Where("decoration_id", decorationId).
		Delete(&UserDecoration{}).Error; err != nil {
		return err
	}

	// 回收也要让用户知道：否则「我的装扮里突然少了一件」无从解释
	SendDecorationNotify(uid, "装扮已被回收："+deco.Name,
		"类型："+DecorationTypeName(deco.Type)+" · 管理员收回了你的这个装扮，如有疑问请联系站点管理员。",
		decorationId)

	return nil
}

// ============================== 站内通知 ==============================

// DecorationExpireWarnDays - 装扮「即将过期」提醒的提前天数（定时任务用）
const DecorationExpireWarnDays = 3

// SendDecorationNotify - 装扮站内通知（统一入口，非事务场景使用）
//
// 所有「获得 / 续期 / 回收 / 即将过期 / 已过期」都通过它落库：
// type=decoration、bind_type=decoration，前台「消息」页据此显示装扮图标并跳转「我的装扮」。
// 站内信无开关（与评论/点赞一致，落库即达），异常只记日志，不影响装扮主流程。
func SendDecorationNotify(uid int, title, content string, decorationId int) {
	CreateUserNotify(uid, 0, NotificationTypeDecoration, title, content, "decoration", decorationId)
}

// decorationSourceName - 获得来源的中文说明（通知正文用）
func decorationSourceName(source string) string {
	switch source {
	case DecorationSourceShop:
		return "商城兑换"
	case DecorationSourceAdmin:
		return "管理员发放"
	case DecorationSourceActivity:
		return "活动获得"
	case DecorationSourceLevel:
		return "等级解锁"
	case DecorationSourceGacha:
		return "抽卡获得"
	case DecorationSourceDefault:
		return "系统默认"
	}
	return source
}

// DecorationExpireText - 有效期文案（0=永久）
func DecorationExpireText(expireTime int64) string {
	if expireTime <= 0 {
		return "永久有效"
	}
	return "至 " + time.Unix(expireTime, 0).Format("2006-01-02 15:04")
}

// decorationNotifyBody - 装扮通知正文
//
// 用「 · 」串成一段而不是多行：前台消息卡片正文只展示两行，
// 多行内容会被截断（有效期、使用提示这类关键信息会看不到）。
func decorationNotifyBody(deco Decoration, source string, expireTime int64) string {
	parts := []string{"类型：" + DecorationTypeName(deco.Type)}
	if !utils.Is.Empty(source) {
		parts = append(parts, "来源："+decorationSourceName(source))
	}
	parts = append(parts, "有效期："+DecorationExpireText(expireTime), "前往「我的装扮」即可佩戴")
	return strings.Join(parts, " · ")
}

// ============================== 佩戴 / 卸下 ==============================

// IsWearingDecoration - 用户是否正在佩戴指定装扮
func IsWearingDecoration(uid int, decorationId int) bool {
	if uid <= 0 || decorationId <= 0 {
		return false
	}
	var user Users
	if err := facade.DB.Drive().Where("id = ?", uid).First(&user).Error; err != nil {
		return false
	}
	for _, value := range wearingMap(user.Json) {
		if cast.ToInt(value) == decorationId {
			return true
		}
	}
	return false
}

// decorationRenderState - 需要写回 users 行的内容
type decorationRenderState struct {
	Json     map[string]any
	Title    string
	SetTitle bool
}

// updateUserDecorationState - 修改 users 的佩戴引用（事务内）
//
// 这里用 UpdateColumns 而非 Update：前者会跳过 GORM Hook，
// 避免误触发 Users.AfterSave 中的头像同步逻辑。
func updateUserDecorationState(tx *gorm.DB, uid int, mutate func(jsonData map[string]any) (*decorationRenderState, error)) error {

	var user Users
	if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
		return errors.New("用户不存在！")
	}

	jsonData := jsonMapOf(user.Json)
	if jsonData == nil {
		jsonData = map[string]any{}
	}
	// 首次进入装扮体系时留存原头衔文本，卸下头衔后可自动恢复
	if _, exist := jsonData["title_text"]; !exist && !utils.Is.Empty(user.Title) {
		jsonData["title_text"] = user.Title
	}

	state, err := mutate(jsonData)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	updates := map[string]any{"json": utils.Json.Encode(state.Json)}
	if state.SetTitle {
		updates["title"] = state.Title
	}
	return tx.Model(&Users{}).Where("id = ?", uid).UpdateColumns(updates).Error
}

// syncWearState - 写入或清除某类型装扮的佩戴状态（deco 为 nil 表示卸下）
//
// users.json.decorations（{type: decoration_id}）是佩戴状态的唯一真相；
// 同时把渲染结果回写到 users.title / users.json.frame 等历史字段，
// 使既有的头像框、头衔展示组件无需改动即可生效。
func syncWearState(tx *gorm.DB, uid int, decoType string, deco *Decoration, text string) error {

	return updateUserDecorationState(tx, uid, func(jsonData map[string]any) (*decorationRenderState, error) {

		wearing := jsonMapOf(jsonData["decorations"])
		if wearing == nil {
			wearing = map[string]any{}
		}

		titleText := ""

		if deco == nil {
			delete(wearing, decoType)
			switch decoType {
			case DecorationTypeAvatarFrame:
				delete(jsonData, "frame")
				delete(jsonData, "frame_scale")
			case DecorationTypeTitle:
				delete(jsonData, "title_decoration")
				titleText = cast.ToString(jsonData["title_text"])
			}
		} else {
			payload := decorationPayloadMap(deco.Payload)
			wearing[decoType] = deco.Id
			switch decoType {
			case DecorationTypeAvatarFrame:
				jsonData["frame"] = cast.ToString(payload["image"])
				if scale := cast.ToFloat64(payload["scale"]); scale > 0 {
					jsonData["frame_scale"] = scale
				}
			case DecorationTypeTitle:
				jsonData["title_decoration"] = map[string]any{
					"id":      deco.Id,
					"name":    deco.Name,
					"payload": payload,
				}
				titleText = text
				if utils.Is.Empty(titleText) {
					titleText = cast.ToString(payload["text"])
				}
			}
		}

		jsonData["decorations"] = wearing

		state := &decorationRenderState{Json: jsonData}
		if decoType == DecorationTypeTitle {
			state.SetTitle = true
			state.Title = titleText
		}
		return state, nil
	})
}

// unwearTypeTx - 事务内卸下某类型正在佩戴的装扮（未佩戴时静默返回）
func unwearTypeTx(tx *gorm.DB, uid int, decoType string) error {

	if utils.Is.Empty(decoType) {
		return nil
	}

	var user Users
	if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
		return errors.New("用户不存在！")
	}
	if wearingByType(user.Json, decoType) <= 0 {
		return nil
	}

	if err := syncWearState(tx, uid, decoType, nil, ""); err != nil {
		return err
	}

	return tx.Model(&UserDecoration{}).
		Where("uid", uid).
		Where("type", decoType).
		Where("status", UserDecorationStatusWearing).
		UpdateColumn("status", UserDecorationStatusOwned).Error
}

// WearDecoration - 佩戴装扮（每种类型同时只能佩戴一个）
func WearDecoration(uid int, decorationId int, text string) (facade.H, error) {

	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	config := GetDecorationConfig()
	if cast.ToInt(config["enabled"]) != 1 {
		return nil, errors.New("装扮功能未开启！")
	}

	var deco Decoration
	if err := facade.DB.Drive().Where("id = ?", decorationId).First(&deco).Error; err != nil {
		return nil, errors.New("装扮不存在！")
	}
	if deco.Status != DecorationStatusOn {
		return nil, errors.New("装扮已下架！")
	}
	if !DecorationTypeSupported(deco.Type) {
		return nil, errors.New("暂不支持的装扮类型！")
	}

	typeConfig := asStringMap(config[deco.Type])
	if typeConfig != nil && cast.ToInt(typeConfig["enabled"]) == 0 {
		return nil, errors.New(DecorationTypeName(deco.Type) + "功能未开启！")
	}

	// 拥有校验：系统默认装扮人人可用，其余必须已拥有且未过期
	if deco.IsDefault != 1 {
		var own UserDecoration
		if err := facade.DB.Drive().
			Where("uid", uid).
			Where("decoration_id", decorationId).
			First(&own).Error; err != nil {
			return nil, errors.New("你还没有这个装扮，先去商城兑换吧！")
		}
		if UserDecorationExpired(own.ExpireTime) {
			_ = facade.DB.Drive().Model(&UserDecoration{}).Where("id = ?", own.Id).
				UpdateColumn("status", UserDecorationStatusExpired).Error
			return nil, errors.New("该装扮已过期！")
		}
	}

	// 头衔文字：装扮自带文字优先，否则使用用户自填文字（纯样式类模板）
	customText := strings.TrimSpace(text)
	if deco.Type == DecorationTypeTitle {
		payload := decorationPayloadMap(deco.Payload)
		if utils.Is.Empty(cast.ToString(payload["text"])) {
			if utils.Is.Empty(customText) {
				return nil, errors.New("请填写头衔文字！")
			}
			maxLength := 10
			if typeConfig != nil && cast.ToInt(typeConfig["max_length"]) > 0 {
				maxLength = cast.ToInt(typeConfig["max_length"])
			}
			if utf8.RuneCountInString(customText) > maxLength {
				return nil, fmt.Errorf("头衔最多 %d 个字！", maxLength)
			}
		} else {
			customText = cast.ToString(payload["text"])
		}
	}

	err := facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 同类型只能佩戴一个：先卸下旧的
		if err := unwearTypeTx(tx, uid, deco.Type); err != nil {
			return err
		}

		// 写入佩戴引用与渲染字段
		if err := syncWearState(tx, uid, deco.Type, &deco, customText); err != nil {
			return err
		}

		// 同步拥有记录的佩戴状态（系统默认装扮不建记录）
		if deco.IsDefault != 1 {
			updates := map[string]any{
				"status":    UserDecorationStatusWearing,
				"wear_time": time.Now().Unix(),
				"type":      deco.Type,
			}
			if !utils.Is.Empty(customText) {
				// 注意：map 形式的更新用的是**数据库列名**（Go 字段是 Text → 列 text），
				// 而不是 json tag。写成 custom_text 会导致
				// Error 1054: Unknown column 'custom_text' in 'field list'。
				updates["text"] = customText
			}
			return tx.Model(&UserDecoration{}).
				Where("uid", uid).
				Where("decoration_id", decorationId).
				UpdateColumns(updates).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return WearingDecorations(uid), nil
}

// UnwearDecoration - 卸下某类型正在佩戴的装扮
func UnwearDecoration(uid int, decoType string) error {

	if uid <= 0 {
		return errors.New("请先登录！")
	}
	if !DecorationTypeSupported(decoType) {
		return errors.New("暂不支持的装扮类型！")
	}

	var user Users
	if err := facade.DB.Drive().Where("id = ?", uid).First(&user).Error; err != nil {
		return errors.New("用户不存在！")
	}
	if wearingByType(user.Json, decoType) <= 0 {
		return errors.New("你还没有佩戴该装扮！")
	}

	return facade.DB.Drive().Transaction(func(tx *gorm.DB) error {
		return unwearTypeTx(tx, uid, decoType)
	})
}

// wearingMap - 从 users.json 中取佩戴引用（json.decorations = {type: decoration_id}）
func wearingMap(jsonValue any) map[string]any {
	jsonData := jsonMapOf(jsonValue)
	if jsonData == nil {
		return map[string]any{}
	}
	wearing := jsonMapOf(jsonData["decorations"])
	if wearing == nil {
		return map[string]any{}
	}
	return wearing
}

// wearingByType - 取某类型正在佩戴的装扮ID（0 表示未佩戴）
func wearingByType(jsonValue any, typ string) int {
	return cast.ToInt(wearingMap(jsonValue)[typ])
}

// WearingDecorations - 用户当前佩戴的装扮（type → 装扮详情）
func WearingDecorations(uid int) facade.H {
	result := facade.H{}
	if uid <= 0 {
		return result
	}

	var user Users
	if err := facade.DB.Drive().Where("id = ?", uid).First(&user).Error; err != nil {
		return result
	}

	wearing := wearingMap(user.Json)
	if len(wearing) == 0 {
		return result
	}

	ids := make([]int, 0, len(wearing))
	for _, value := range wearing {
		if id := cast.ToInt(value); id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return result
	}

	var decoList []Decoration
	list, _ := facade.DB.Model(&decoList).Where("id", "in", ids).Select()

	// 自填文字（如纯样式类头衔）：把用户保存的文本一并带回。
	// 前端「当前佩戴」行读的是 custom_text（Select() 的结果按 json tag 命名），
	// 不带上它时样式类头衔会渲染成空徽章。
	textMap := map[int]string{}
	var ownList []UserDecoration
	if owns, err := facade.DB.Model(&ownList).
		Where("uid", uid).
		Where("decoration_id", "in", ids).
		Select(); err == nil {
		for _, own := range owns {
			if text := cast.ToString(own["custom_text"]); !utils.Is.Empty(text) {
				textMap[cast.ToInt(own["decoration_id"])] = text
			}
		}
	}

	for _, item := range list {
		typ := cast.ToString(item["type"])
		if utils.Is.Empty(typ) {
			continue
		}
		item["payload"] = decorationPayloadMap(item["payload"])
		if text, exist := textMap[cast.ToInt(item["id"])]; exist {
			item["custom_text"] = text
		}
		result[typ] = item
	}
	return result
}
