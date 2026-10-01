package model

import (
	"errors"
	"inis/app/facade"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// ============================== 卡密奖励资产（纯卡密 · 库存制） ==============================
//
// 奖励项里自己填卡密内容，发出去的就是这些码：
//
//	{"asset":"card","codes":["SN2026000001","SN2026000002"]}
//
// 语义：
//   - **与积分无关**：卡密就是一张码，没有面额，也不进「积分 → 卡密」的池子；
//     用户拿到码之后自己去用（外部渠道兑换、线下核销…），本站不做二次兑换；
//   - **库存制**：codes 就是库存清单，每发一次消耗一张（见 reward-card-stock.go），
//     **发完即失效** —— 没有未发放的码时这次奖励什么都不发；
//   - **不降级**：库存不足不改发积分 / 经验，也不会让签到失败，只在奖励明细里标记
//     card_missing（前端据此提示「卡密已发完」）；
//   - 数值字段（value）不参与卡密逻辑，只是为了满足奖励引擎「value / min / max 全 0
//     的奖励项会被丢弃」的规则（见 reward.go 的 parseRewardItem），后台会自动写成 1（1 张）。

// RewardCardAssetKey - 卡密奖励的资产标识（各处判断「这条奖励是不是卡密」用）
const RewardCardAssetKey = "card"

func init() {
	RegisterRewardAsset(RewardAsset{
		Key:  RewardCardAssetKey,
		Name: "卡密",
		Unit: "张",
		Icon: "bi-ticket-perforated",
		Desc: "卡密内容在奖励项里自己填（一行一个），与积分无关；库存发完即失效，不改发别的奖励",
		Grant: func(tx *gorm.DB, uid int, value int, meta facade.H) (facade.H, error) {

			codes := ParseCardCodes(asStringMap(meta["config"])["codes"])

			// 没配置卡密内容 = 没有库存：什么都不发（不降级）
			if len(codes) == 0 {
				facade.Log.Warn(map[string]any{
					"uid":  uid,
					"type": cast.ToString(meta["type"]),
				}, "卡密奖励未配置卡密内容，本次不发放")
				return facade.H{"card_missing": true, "reason": "未配置卡密"}, nil
			}

			extra, err := GrantRewardCardTx(tx, uid, codes)
			if err == nil {
				return extra, nil
			}

			// 库存发完：这是正常状态，不发任何替代奖励
			if errors.Is(err, ErrRewardCardEmpty) {
				facade.Log.Info(map[string]any{
					"uid":  uid,
					"type": cast.ToString(meta["type"]),
				}, "卡密库存已发完，本次不发放")
				return facade.H{"card_missing": true, "reason": "库存已发完"}, nil
			}

			return nil, err
		},
		Balance: func(uid int) int {
			// 库存按「奖励项里填的卡密」计算，资产级没有统一余额
			// （后台用 checkin/card-stock 按奖励项查询，见 RewardCardStock）
			return 0
		},
	})
}
