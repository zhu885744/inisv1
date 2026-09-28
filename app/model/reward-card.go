package model

import (
	"errors"
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

// ============================== 卡密奖励资产 ==============================
//
// 把「积分卡密」接进奖励引擎：任何奖励配置（签到基础奖励 / 周期奖励 / 里程碑 / 月全勤 …）
// 里写 {"asset":"card","value":100} 就能发一张面额 100 的卡密。
//
// 发放语义：从卡密池取一张「未使用 + 未过期」的卡密 → 标记为「已发放」并绑定用户
// （见 model/integral-card.go 的 GrantIntegralCardTx），卡密明文会随奖励明细一起返回，
// 用户到「积分 → 卡密兑换」把它兑换成积分。
//
// 池子为空时不能把整次签到搞失败，因此在资产内部做降级：
//
//	{"asset":"card","value":100,"fallback":"integral"}          没卡了 → 改发 100 积分（默认策略）
//	{"asset":"card","value":100,"fallback":"none"}              没卡了 → 不发，只在明细里标注
//	{"asset":"card","value":100,"fallback":"integral","fallback_value":50}  改发 50 积分
//
// value 填 0 表示「不限面额」（取池子里任意一张）。

// RewardCardAssetKey - 卡密奖励的资产标识（各处判断「这条奖励是不是卡密」用）
const RewardCardAssetKey = "card"

// RewardCardFallbackNone - 卡密不足时「什么都不发」的降级标记
const RewardCardFallbackNone = "none"

func init() {
	RegisterRewardAsset(RewardAsset{
		Key:  RewardCardAssetKey,
		Name: "卡密",
		Unit: "张",
		Icon: "bi-ticket-perforated",
		Desc: "数值 = 卡密面额（0 表示任意面额）；卡密池不足时按 fallback 降级发放",
		Grant: func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {

			extra, err := GrantIntegralCardTx(tx, uid, value, meta)
			if err == nil {
				return extra, nil
			}
			if !errors.Is(err, ErrIntegralCardEmpty) {
				return nil, err
			}

			// 池子为空：按 fallback 降级，不让签到整体失败
			return fallbackRewardCard(tx, uid, value, meta)
		},
		Balance: func(uid int) int {
			return AvailableIntegralCardCount(0)
		},
	})
}

// fallbackRewardCard - 卡密池为空时的降级发放
func fallbackRewardCard(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {

	config := asStringMap(meta["config"])

	fallback := cast.ToString(config["fallback"])
	if utils.Is.Empty(fallback) {
		// 默认改发等额积分：对用户最友好，也最接近卡密本身的价值
		fallback = "integral"
	}

	result := facade.H{
		"card_missing": true,
		"fallback":     fallback,
	}

	if fallback == RewardCardFallbackNone {
		facade.Log.Warn(map[string]any{"uid": uid, "value": value}, "卡密池为空，本次卡密奖励已跳过（fallback=none）")
		return result, nil
	}

	grant := RewardAssetWithGrant(fallback)
	if grant == nil {
		facade.Log.Error(map[string]any{"fallback": fallback, "uid": uid}, "卡密降级资产未注册，本次卡密奖励已跳过")
		return result, nil
	}

	// 降级数量：优先 fallback_value，缺省用卡密面额；都为 0 时无法降级
	amount := cast.ToInt(config["fallback_value"])
	if amount <= 0 {
		amount = value
	}
	if amount <= 0 {
		facade.Log.Warn(map[string]any{"uid": uid}, "卡密池为空且未配置降级数量，本次卡密奖励已跳过")
		return result, nil
	}

	extra, err := grant(tx, uid, amount, meta)
	if err != nil {
		return nil, err
	}

	result["fallback_value"] = amount
	if len(extra) > 0 {
		result["fallback_extra"] = extra
	}

	facade.Log.Warn(map[string]any{
		"uid":      uid,
		"fallback": fallback,
		"amount":   amount,
	}, "卡密池为空，已按降级策略发放")

	return result, nil
}
