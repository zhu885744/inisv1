package model

import (
	"fmt"
	"inis/app/facade"
	"reflect"
	"sort"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

// ============================== 奖励引擎 ==============================
//
// 背景：站点里的「奖励」原先只有两种硬编码的发放方式 —— 经验走 EXP.Add、积分走 Integral.Add，
// 两者互不相干，调用方需要自己成对调用（签到就是手动串了两次）。想要「一次活动发多种奖励」
// （经验 + 积分 + 金币 / 优惠券 / 会员天数 …）就得每次改业务代码。
//
// 这里抽出一层「奖励引擎」：
//  1. 奖励项（RewardItem）用配置描述：资产类型 + 数值（或随机区间 / 触发概率）；
//  2. 资产（RewardAsset）以注册表的形式挂载：谁想支持一种新奖励，实现 Grant / Balance 后
//     RegisterRewardAsset 注册一次即可，业务侧不需要改动（见文件末尾的内置 exp / integral）；
//  3. 发放（GrantRewards）在**同一个事务**里逐项落库，失败整体回滚，避免「经验加了积分没加」。
//
// 签到（checkin.go）是第一个使用者，之后的活动 / 任务 / 抽奖等都可以复用。

// RewardAsset - 一种奖励资产（可扩展）
//
// Key    资产标识，配置里写它，如 exp / integral / card / coin；
// Name   展示名，如「经验」；
// Unit   单位，如「经验值」「积分」；
// Icon   Bootstrap Icons 类名，仅用于前端展示；
// Desc   后台配置页的输入提示（例如卡密要填面额），可为空；
// Grant  发放实现：value 为正表示发放、为负表示扣除（实现方需自行校验余额）；
// Balance 查询余额，用于展示 / 事务内校验，取不到时返回 0。
//
// Grant 返回的 map 是「发放结果附加信息」（如卡密的明文与面额），引擎会把它回填到奖励明细的
// extra 字段里，不需要附加信息时返回 nil 即可。
//
// 说明：Grant 会收到调用方传入的 meta，常见键有 type / description / json，
// 以及 config（该奖励项在配置里的额外键，如卡密的 fallback），按需读取即可。
type RewardAsset struct {
	Key     string
	Name    string
	Unit    string
	Icon    string
	Desc    string
	Grant   func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error)
	Balance func(uid int) int
}

var (
	rewardAssets   = make(map[string]RewardAsset)
	rewardAssetSeq = make([]string, 0, 4)
	rewardAssetMu  sync.RWMutex
)

// RegisterRewardAsset - 注册（或覆盖）一种奖励资产
//
// 二次开发示例（金币资产）：
//
//	model.RegisterRewardAsset(model.RewardAsset{
//		Key: "coin", Name: "金币", Unit: "金币", Icon: "bi-coin",
//		Grant: func(tx *gorm.DB, uid, value int, meta facade.H) error {
//			return tx.Model(&model.Users{}).Where("id", uid).
//				UpdateColumn("json", ...).Error // 自行实现存储
//		},
//		Balance: func(uid int) int { return 0 },
//	})
//
// 注册后，任何奖励配置（签到基础奖励 / 随机奖励 / 里程碑 …）里写 {"asset":"coin","value":100}
// 即可自动生效，接口返回的资产清单也会带上它，前端无需改动。
func RegisterRewardAsset(item RewardAsset) {
	if utils.Is.Empty(item.Key) {
		return
	}
	rewardAssetMu.Lock()
	defer rewardAssetMu.Unlock()

	if _, exist := rewardAssets[item.Key]; !exist {
		rewardAssetSeq = append(rewardAssetSeq, item.Key)
	}
	rewardAssets[item.Key] = item
}

// RewardAssetOf - 按标识取资产
func RewardAssetOf(key string) (RewardAsset, bool) {
	rewardAssetMu.RLock()
	defer rewardAssetMu.RUnlock()

	item, exist := rewardAssets[key]
	return item, exist
}

// RewardAssetList - 全部已注册资产（按注册顺序）
func RewardAssetList() []RewardAsset {
	rewardAssetMu.RLock()
	defer rewardAssetMu.RUnlock()

	result := make([]RewardAsset, 0, len(rewardAssetSeq))
	for _, key := range rewardAssetSeq {
		result = append(result, rewardAssets[key])
	}
	return result
}

// RewardAssetOptions - 资产清单（接口返回给前端用：标识 / 名称 / 单位 / 图标）
func RewardAssetOptions() []facade.H {
	result := make([]facade.H, 0)
	for _, item := range RewardAssetList() {
		result = append(result, facade.H{
			"key":  item.Key,
			"name": item.Name,
			"unit": item.Unit,
			"icon": item.Icon,
			"desc": item.Desc,
		})
	}
	return result
}

// RewardAssetWithGrant - 取资产的发放实现（未注册时返回 nil，由调用方决定降级）
func RewardAssetWithGrant(key string) func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {
	if asset, exist := RewardAssetOf(key); exist {
		return asset.Grant
	}
	return nil
}

// RewardItem - 一条奖励项（配置里的最小单位）
//
//	{"asset":"exp","value":10}                          固定 10 经验
//	{"asset":"integral","min":5,"max":15}               5~15 积分（随机）
//	{"asset":"integral","value":20,"chance":10}         10% 概率获得 20 积分
//	{"exp":10,"integral":5}                             简洁写法（键即资产标识）
type RewardItem struct {
	Asset  string   // 资产标识
	Value  int      // 固定值（与 Min/Max 二选一）
	Min    int      // 随机下限
	Max    int      // 随机上限
	Chance int      // 触发概率（0 或 >=100 表示必得）
	Label  string   // 展示标签，如「幸运奖励」
	Group  string   // 来源分组：base / cycle / streak / milestone / monthly / random
	Text   string   // 补充说明（如「连续 7 天」）
	Extra  facade.H // 该奖励项的额外配置（asset/value/label/chance 之外的键，原样交给资产实现）
}

// asStringMap - 把任意 map 形态的值转成 map[string]any
//
// 为什么需要它：配置里的对象有两种来源 —— 数据库 JSON 解码得到 map[string]any，
// 而代码里的默认配置（facade.H{...}）是**命名类型**。cast.ToStringMap 用的是类型断言，
// 命名类型不在断言目标里，会静默返回空 map，导致「默认配置读不到、奖励算成 0」。
// 这里先覆盖两种常见类型，其余 map 类型走反射兜底。
func asStringMap(value any) map[string]any {
	if value == nil {
		return nil
	}

	switch item := value.(type) {
	case map[string]any:
		return item
	case facade.H:
		return map[string]any(item)
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return nil
	}

	result := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		result[fmt.Sprintf("%v", key.Interface())] = rv.MapIndex(key).Interface()
	}
	return result
}

// ParseRewardItems - 把配置里的奖励项解析成 RewardItem 列表
//
// 兼容三种写法（同一份配置里可以混用）：
//  1. 数组：[{"asset":"exp","value":10}, {"asset":"integral","value":5}]
//  2. 简洁对象：{"exp":10,"integral":5}（键是资产标识，值是数值）
//  3. 单个对象：{"asset":"exp","value":10}
func ParseRewardItems(raw any, group string) []RewardItem {
	result := make([]RewardItem, 0)

	switch value := raw.(type) {
	case nil:
		return result
	case []any:
		for _, item := range value {
			result = append(result, parseRewardItem(asStringMap(item), group)...)
		}
		return result
	case []facade.H:
		for _, item := range value {
			result = append(result, parseRewardItem(asStringMap(item), group)...)
		}
		return result
	default:
		return parseRewardItem(asStringMap(raw), group)
	}
}

// parseRewardItem - 解析单个奖励项（可能是简洁写法，因此返回切片）
func parseRewardItem(row map[string]any, group string) []RewardItem {
	if len(row) == 0 {
		return nil
	}

	// 标准写法：带 asset 字段
	if asset := cast.ToString(row["asset"]); !utils.Is.Empty(asset) {
		item := RewardItem{
			Asset:  asset,
			Value:  cast.ToInt(row["value"]),
			Min:    cast.ToInt(row["min"]),
			Max:    cast.ToInt(row["max"]),
			Chance: cast.ToInt(row["chance"]),
			Label:  cast.ToString(row["label"]),
			Group:  group,
			Text:   cast.ToString(row["text"]),
			Extra:  rewardItemExtra(row),
		}
		if utils.Is.Empty(item.Label) {
			item.Label = cast.ToString(row["name"])
		}
		if item.Value == 0 && item.Min == 0 && item.Max == 0 {
			return nil
		}
		return []RewardItem{item}
	}

	// 简洁写法：{"exp":10,"integral":5}，键为资产标识
	result := make([]RewardItem, 0, len(row))
	for key, value := range row {
		if _, exist := RewardAssetOf(key); !exist {
			continue
		}
		number := cast.ToInt(value)
		if number == 0 {
			continue
		}
		result = append(result, RewardItem{
			Asset: key,
			Value: number,
			Group: group,
		})
	}
	// 保证输出顺序稳定（map 遍历无序）
	sort.SliceStable(result, func(i, j int) bool { return result[i].Asset < result[j].Asset })
	return result
}

// rewardItemExtra - 取出奖励项里「通用字段之外」的配置，原样交给资产实现
//
// 例如卡密可以写 {"asset":"card","value":100,"fallback":"integral"}，
// fallback 就是额外配置，卡密资产在池子为空时按它降级。
func rewardItemExtra(row map[string]any) facade.H {
	extra := facade.H{}

	for key, value := range row {
		switch key {
		case "asset", "value", "min", "max", "chance", "label", "name", "text":
			continue
		}
		extra[key] = value
	}

	if len(extra) == 0 {
		return nil
	}
	return extra
}

// RollRewardItems - 结算随机性与概率，得到「本次实际发放」的奖励项
//
//   - Chance 在 (0,100) 之间时按概率决定是否发放；
//   - Min/Max 都大于 0 时在区间内取随机值（Min > Max 时自动交换，取不到范围时退化为固定值）。
func RollRewardItems(items []RewardItem) []RewardItem {
	result := make([]RewardItem, 0, len(items))

	for _, item := range items {
		if item.Chance > 0 && item.Chance < 100 {
			if utils.Rand.Int(100, 1) > item.Chance {
				continue
			}
		}

		min, max := item.Min, item.Max
		if min > 0 && max > 0 {
			if min > max {
				min, max = max, min
			}
			item.Value = utils.Rand.Int(min, max)
		}

		if item.Value <= 0 {
			continue
		}

		result = append(result, item)
	}

	return result
}

// SplitRewardItems - 按「是否必得」拆分奖励项
//
// 返回 sure：必得的固定奖励；variable：不确定的奖励（带概率 chance，或只配了随机区间 min/max）。
//
// 为什么需要它：预览（未签到时的「签到可得」）如果把概率项也算进合计，用户会误以为必得 ——
// 例如「签到可得 22 积分」里其实有 20 分是 10% 概率。前端应把 variable 单独用
// 「另有 10% 概率获得 20 积分」这类文案展示，不要混进合计。
func SplitRewardItems(items []RewardItem) (sure []RewardItem, variable []RewardItem) {
	sure = make([]RewardItem, 0, len(items))
	variable = make([]RewardItem, 0)

	for _, item := range items {
		// 概率奖励：可能不发
		if item.Chance > 0 && item.Chance < 100 {
			variable = append(variable, item)
			continue
		}
		// 随机区间：金额不确定（Value 为 0），只有上下限
		if item.Value <= 0 && item.Max > 0 {
			variable = append(variable, item)
			continue
		}
		sure = append(sure, item)
	}

	return
}

// GroupRewardItems - 按资产汇总奖励项（用于展示「共获得多少经验 / 积分」）
func GroupRewardItems(items []RewardItem) (total facade.H, detail []facade.H) {
	total = facade.H{}
	detail = make([]facade.H, 0, len(items))

	for _, item := range items {
		total[item.Asset] = cast.ToInt(total[item.Asset]) + item.Value
		detail = append(detail, RewardItemToMap(item))
	}

	return
}

// RewardItemToMap - 奖励项 → 展示用结构（含资产名称 / 单位 / 图标，前端无需再查表）
func RewardItemToMap(item RewardItem) facade.H {
	asset, exist := RewardAssetOf(item.Asset)

	result := facade.H{
		"asset": item.Asset,
		"value": item.Value,
		"label": item.Label,
		"group": item.Group,
		"text":  item.Text,
		"name":  item.Asset,
		"unit":  item.Asset,
		"icon":  "bi-gift",
	}
	if exist {
		result["name"] = asset.Name
		result["unit"] = asset.Unit
		if !utils.Is.Empty(asset.Icon) {
			result["icon"] = asset.Icon
		}
	}
	// 概率 / 区间信息原样带上，供前端展示「有 10% 概率额外获得」
	if item.Chance > 0 {
		result["chance"] = item.Chance
	}
	if item.Min > 0 || item.Max > 0 {
		result["min"] = item.Min
		result["max"] = item.Max
	}

	return result
}

// GrantRewards - 发放奖励（自建事务）
//
// items 视为「已确定」的奖励项：带概率 / 区间的配置请先取 RollRewardItems 结算，
// 这样发放结果与实际入账一致，业务侧也能把同一份结果写进业务记录。
// 未知资产会被跳过并记日志（不影响其它奖励）。
// meta 常用键：type（流水类型，如 check-in）、description（流水描述）、json（附加信息）。
func GrantRewards(uid int, items []RewardItem, meta facade.H) (facade.H, error) {
	if uid <= 0 {
		return nil, fmt.Errorf("请先登录！")
	}
	return GrantRewardsTx(nil, uid, items, meta)
}

// GrantRewardsTx - 在指定事务内发放奖励
//
// tx 为 nil 时自建事务（普通调用用 GrantRewards 即可）；签到这类「写业务记录 + 发奖励」
// 需要原子的场景，把外层事务传进来即可。
func GrantRewardsTx(tx *gorm.DB, uid int, items []RewardItem, meta facade.H) (facade.H, error) {
	total, detail := GroupRewardItems(items)

	result := facade.H{
		"items": detail,
		"total": total,
	}

	if len(items) == 0 {
		return result, nil
	}

	grant := func(tx *gorm.DB) error {
		for index, item := range items {
			asset, exist := RewardAssetOf(item.Asset)
			if !exist || asset.Grant == nil {
				facade.Log.Error(map[string]any{"asset": item.Asset, "uid": uid}, "奖励资产未注册，已跳过")
				continue
			}

			// 把该奖励项的额外配置透传给资产实现（如卡密的 fallback）
			itemMeta := make(facade.H, len(meta)+1)
			for key, value := range meta {
				itemMeta[key] = value
			}
			if len(item.Extra) > 0 {
				itemMeta["config"] = item.Extra
			}

			extra, err := asset.Grant(tx, uid, item.Value, itemMeta)
			if err != nil {
				return err
			}

			// 发放结果附加信息（如卡密明文）回填到明细，供记录与前端展示
			if len(extra) > 0 && index < len(detail) {
				detail[index]["extra"] = extra
			}
		}
		return nil
	}

	if tx != nil {
		return result, grant(tx)
	}

	err := facade.DB.Drive().Transaction(func(tx *gorm.DB) error {
		return grant(tx)
	})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid, "total": total}, "奖励发放失败")
	}

	return result, err
}

// ============================== 内置资产 ==============================

func init() {
	// 经验：写 inis_exp 流水 + users.exp 自增
	RegisterRewardAsset(RewardAsset{
		Key:  "exp",
		Name: "经验",
		Unit: "经验值",
		Icon: "bi-star",
		Desc: "数值为经验值",
		Grant: func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {
			return nil, grantExpTx(tx, uid, value, meta)
		},
		Balance: func(uid int) int {
			user, _ := facade.DB.Model(&Users{}).Where("id", uid).Find()
			return cast.ToInt(user["exp"])
		},
	})

	// 积分：写 inis_integral 流水 + users.integral 自增（含余额快照）
	RegisterRewardAsset(RewardAsset{
		Key:  "integral",
		Name: "积分",
		Unit: "积分",
		Icon: "bi-coin",
		Desc: "数值为积分数量",
		Grant: func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {
			return nil, grantIntegralTx(tx, uid, value, meta)
		},
		Balance: IntegralBalance,
	})
}

// grantExpTx - 经验流水 + 用户经验值自增（同一事务）
func grantExpTx(tx *gorm.DB, uid int, value int, meta facade.H) error {
	if value == 0 {
		return nil
	}

	table := EXP{
		Uid:         uid,
		Value:       value,
		Type:        cast.ToString(meta["type"]),
		BindType:    cast.ToString(meta["bind_type"]),
		BindId:      cast.ToInt(meta["bind_id"]),
		Description: cast.ToString(meta["description"]),
	}
	if utils.Is.Empty(table.Type) {
		table.Type = "reward"
	}
	if utils.Is.Empty(table.Description) {
		table.Description = "奖励"
	}
	if meta["json"] != nil {
		table.Json = utils.Json.Encode(meta["json"])
	}

	if err := tx.Create(&table).Error; err != nil {
		return err
	}

	return tx.Model(&Users{}).Where("id", uid).UpdateColumn("exp", gorm.Expr("exp + ?", value)).Error
}

// grantIntegralTx - 积分流水 + 用户积分余额自增（同一事务）
//
// 注意：这里不走 Integral.Add —— 那个入口会按「积分规则」取固定值并做每日上限校验，
// 而奖励引擎发放的是「配置里算出来的真实数额」（签到奖励、随机奖励、里程碑 …）。
// 每日的次数约束由业务侧（如签到记录的唯一索引）保证。
func grantIntegralTx(tx *gorm.DB, uid int, value int, meta facade.H) error {
	if value == 0 {
		return nil
	}

	table := Integral{
		Uid:         uid,
		Value:       value,
		Type:        cast.ToString(meta["type"]),
		Description: cast.ToString(meta["description"]),
	}
	if utils.Is.Empty(table.Type) {
		table.Type = "reward"
	}
	if utils.Is.Empty(table.Description) {
		table.Description = "奖励"
	}
	if meta["json"] != nil {
		table.Json = utils.Json.Encode(meta["json"])
	} else {
		table.Json = utils.Json.Encode(map[string]any{
			"balance_after": IntegralBalance(uid) + value,
		})
	}

	// 扣除（value < 0）：用带条件的原子更新校验余额，避免并发下扣成负数
	if value < 0 {
		result := tx.Model(&Users{}).Where("id", uid).Where("integral >= ?", -value).
			UpdateColumn("integral", gorm.Expr("integral + ?", value))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("积分不足！")
		}
		if err := tx.Create(&table).Error; err != nil {
			return err
		}
		return nil
	}

	if err := tx.Create(&table).Error; err != nil {
		return err
	}

	return tx.Model(&Users{}).Where("id", uid).UpdateColumn("integral", gorm.Expr("integral + ?", value)).Error
}
