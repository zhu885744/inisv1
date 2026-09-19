package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 商品状态常量
const (
	GoodsStatusOff = 0 // 下架
	GoodsStatusOn  = 1 // 上架
)

// 订单状态常量
const (
	OrderStatusPending   = 0 // 待发货
	OrderStatusShipped   = 1 // 已发货
	OrderStatusCompleted = 2 // 已完成
	OrderStatusCanceled  = 3 // 已取消（积分已退还、库存已回滚）
)

// 商品类型常量
const (
	GoodsTypeVirtual  = "virtual"  // 虚拟商品
	GoodsTypePhysical = "physical" // 实物商品
)

// 虚拟商品发货方式常量
const (
	DeliverText = "text" // 文本发货
	DeliverCard = "card" // 卡密发货
)

// Goods - 商品表
type Goods struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Title       string `gorm:"size:128; comment:商品名称;" json:"title"`
	Description string `gorm:"type:text; comment:商品描述; default:Null;" json:"description"`
	Cover       string `gorm:"comment:商品封面; default:Null;" json:"cover"`
	Price       int    `gorm:"type:int(32); comment:积分价格; default:0;" json:"price"`
	Stock       int    `gorm:"type:int(32); comment:库存; default:0;" json:"stock"`
	Status      int    `gorm:"tinyint; default:1; comment:状态（0下架 1上架）;" json:"status"`
	Type        string `gorm:"size:16; comment:商品类型（virtual虚拟 physical实物）; default:'virtual';" json:"type"`
	DeliverType string `gorm:"size:16; comment:发货方式（text文本 card卡密，仅虚拟商品）; default:'';" json:"deliver_type"`
	// 商城增强字段
	Category     string `gorm:"size:32; index; comment:商品分类（自由字符串，用于前台分组）; default:Null;" json:"category"`
	LimitPerUser int    `gorm:"type:int(32); comment:每人限购数量（0=不限购）; default:0;" json:"limit_per_user"`
	MinExp       int    `gorm:"type:int(32); comment:兑换所需最低经验值（0=不限）; default:0;" json:"min_exp"`
	StartTime    int64  `gorm:"comment:兑换开始时间（0=不限）; default:0;" json:"start_time"`
	EndTime      int64  `gorm:"comment:兑换结束时间（0=不限）; default:0;" json:"end_time"`
	Sort         int    `gorm:"type:int(32); index; comment:排序权重（越大越靠前）; default:0;" json:"sort"`
	Sold         int    `gorm:"type:int(32); comment:已兑换数量; default:0;" json:"sold"`
	// 敏感字段：卡密池、文本发货内容（仅管理员可见）
	DeliverContent string `gorm:"type:text; comment:文本发货内容（text类型）; default:Null;" json:"deliver_content"`
	Cards          string `gorm:"type:longtext; comment:卡密池（JSON数组，card类型）; default:Null;" json:"cards"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// GoodsOrder - 商品订单表
type GoodsOrder struct {
	Id             int    `gorm:"type:int(32); comment:主键;" json:"id"`
	OrderNo        string `gorm:"size:32; uniqueIndex:uk_goods_order_no; comment:订单号;" json:"order_no"`
	Uid            int    `gorm:"type:int(32); index; comment:用户ID;" json:"uid"`
	GoodsId        int    `gorm:"type:int(32); index; comment:商品ID;" json:"goods_id"`
	Price          int    `gorm:"type:int(32); comment:成交积分价格; default:0;" json:"price"`
	Status         int    `gorm:"tinyint; default:0; comment:状态（0待发货 1已发货 2已完成 3已取消）;" json:"status"`
	DeliverContent string `gorm:"type:text; comment:发货内容（虚拟商品：文本/卡密）; default:Null;" json:"deliver_content"`
	Address        string `gorm:"type:text; comment:收货地址（JSON，实物商品）; default:Null;" json:"address"`
	Logistics      string `gorm:"type:text; comment:物流信息（实物商品发货）; default:Null;" json:"logistics"`
	// 下单快照：商品改名/改图后订单仍能正确展示
	GoodsTitle string `gorm:"size:128; comment:商品名称快照;" json:"goods_title"`
	GoodsCover string `gorm:"comment:商品封面快照;" json:"goods_cover"`
	// 取消与完成
	Refund     int   `gorm:"type:int(32); comment:已退还积分; default:0;" json:"refund"`
	CancelTime int64 `gorm:"comment:取消时间; default:0;" json:"cancel_time"`
	FinishTime int64 `gorm:"comment:完成时间; default:0;" json:"finish_time"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitGoods - 初始化商品与订单表
func InitGoods() {
	if err := facade.DB.Drive().AutoMigrate(&Goods{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Goods表迁移失败")
		return
	}
	if err := facade.DB.Drive().AutoMigrate(&GoodsOrder{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "GoodsOrder表迁移失败")
		return
	}
	// 索引由结构体 tag 声明（category / sort / uid / goods_id / order_no），AutoMigrate 会自动创建。
	// 注意：不要用 "CREATE INDEX IF NOT EXISTS"，MySQL 不支持该语法（会静默失败）。
}

// AfterFind - 查询Hook
func (this *Goods) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// AfterFind - 查询Hook
func (this *GoodsOrder) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	this.Result = this.result()
	return
}

// GenerateOrderNo - 生成订单号（时间 + 用户 + 随机数，配合唯一索引防重）
func GenerateOrderNo(uid int) string {
	return fmt.Sprintf(
		"G%s%04d%04d",
		time.Now().Format("20060102150405"),
		uid%10000,
		utils.Rand.Int(0, 9999),
	)
}

// CountUserBought - 统计某用户对某商品的已兑换数量（不含已取消订单）
func CountUserBought(uid int, goodsId int) int64 {
	count, _ := facade.DB.Model(&GoodsOrder{}).
		Where("uid", uid).
		Where("goods_id", goodsId).
		Where("status", "!=", OrderStatusCanceled).
		Count()
	return count
}

// CanBuy - 兑换前校验
// 返回：是否可兑换、不可兑换原因、限购剩余数量（-1 表示不限购）
func (this *Goods) CanBuy(uid int, goods *Goods) (bool, string, int) {
	remain := -1

	// 上架状态
	if goods.Status != GoodsStatusOn {
		return false, "商品已下架！", remain
	}

	// 兑换时间窗口
	now := time.Now().Unix()
	if goods.StartTime > 0 && now < goods.StartTime {
		return false, "兑换尚未开始！", remain
	}
	if goods.EndTime > 0 && now > goods.EndTime {
		return false, "兑换活动已结束！", remain
	}

	// 库存
	if goods.Stock <= 0 {
		return false, "商品库存不足！", remain
	}

	// 卡密池
	if goods.Type == GoodsTypeVirtual && goods.DeliverType == DeliverCard && utils.Is.Empty(goods.Cards) {
		return false, "卡密库存不足！", remain
	}

	if uid <= 0 {
		// 未登录只校验商品维度
		return true, "", remain
	}

	// 经验门槛与积分校验
	user, _ := facade.DB.Model(&Users{}).Where("id", uid).Find()
	if utils.Is.Empty(user) {
		return false, "用户不存在！", remain
	}
	if goods.MinExp > 0 && cast.ToInt(user["exp"]) < goods.MinExp {
		return false, fmt.Sprintf("经验值达到 %d 才可兑换！", goods.MinExp), remain
	}
	if cast.ToInt(user["integral"]) < goods.Price {
		return false, "积分不足！", remain
	}

	// 每人限购
	if goods.LimitPerUser > 0 {
		bought := int(CountUserBought(uid, goods.Id))
		if bought >= goods.LimitPerUser {
			return false, "已达每人限购上限！", 0
		}
		remain = goods.LimitPerUser - bought
	}

	return true, "", remain
}

// Buy - 购买商品（事务：校验 → 扣库存 → 扣积分 → 写流水 → 生成订单 → 虚拟商品立即发货）
// address: 收货地址 JSON 字符串（实物商品必填）
func (this *Goods) Buy(uid int, goodsId int, address string) (order GoodsOrder, err error) {

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 1. 查询商品
		var goods Goods
		if err := tx.Where("id = ?", goodsId).First(&goods).Error; err != nil {
			return errors.New("商品不存在！")
		}
		if goods.Status != GoodsStatusOn {
			return errors.New("商品已下架！")
		}

		// 2. 兑换时间窗口
		now := time.Now().Unix()
		if goods.StartTime > 0 && now < goods.StartTime {
			return errors.New("兑换尚未开始！")
		}
		if goods.EndTime > 0 && now > goods.EndTime {
			return errors.New("兑换活动已结束！")
		}

		// 3. 实物商品校验收货地址
		if goods.Type == GoodsTypePhysical && utils.Is.Empty(address) {
			return errors.New("请填写收货地址！")
		}

		// 4. 卡密商品校验卡密池
		if goods.Type == GoodsTypeVirtual && goods.DeliverType == DeliverCard && utils.Is.Empty(goods.Cards) {
			return errors.New("卡密库存不足！")
		}

		// 5. 校验用户（经验门槛 / 限购）
		var user Users
		if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
			return errors.New("用户不存在！")
		}
		if goods.MinExp > 0 && user.Exp < goods.MinExp {
			return fmt.Errorf("经验值达到 %d 才可兑换！", goods.MinExp)
		}
		if goods.LimitPerUser > 0 {
			var bought int64
			if err := tx.Model(&GoodsOrder{}).
				Where("uid = ? AND goods_id = ? AND status != ?", uid, goodsId, OrderStatusCanceled).
				Count(&bought).Error; err != nil {
				return err
			}
			if bought >= int64(goods.LimitPerUser) {
				return errors.New("已达每人限购上限！")
			}
		}

		// 6. 校验并扣减库存（原子操作，防止超卖）
		result := tx.Model(&Goods{}).Where("id = ? AND stock > 0", goodsId).
			UpdateColumn("stock", gorm.Expr("stock - 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("商品库存不足！")
		}

		// 7. 校验积分余额并扣减
		if user.Integral < goods.Price {
			return errors.New("积分不足！")
		}

		if err := tx.Model(&Users{}).Where("id = ?", uid).
			UpdateColumn("integral", gorm.Expr("integral - ?", goods.Price)).Error; err != nil {
			return err
		}

		// 8. 销量 +1
		if err := tx.Model(&Goods{}).Where("id = ?", goodsId).
			UpdateColumn("sold", gorm.Expr("sold + 1")).Error; err != nil {
			return err
		}

		// 9. 写入积分流水（含余额快照）
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       -goods.Price,
			Type:        IntegralTypeBuy,
			Description: "兑换商品：" + goods.Title,
			Json: utils.Json.Encode(map[string]any{
				"goods_id":      goodsId,
				"balance_after": user.Integral - goods.Price,
			}),
		}).Error; err != nil {
			return err
		}

		// 10. 生成订单（含商品快照）
		order = GoodsOrder{
			OrderNo:    GenerateOrderNo(uid),
			Uid:        uid,
			GoodsId:    goodsId,
			Price:      goods.Price,
			Status:     OrderStatusPending,
			GoodsTitle: goods.Title,
			GoodsCover: goods.Cover,
		}

		// 11. 根据商品类型处理发货
		if goods.Type == GoodsTypePhysical {
			// 实物商品：保存收货地址，等待管理员发货
			order.Address = address
		} else {
			// 虚拟商品：立即发货
			switch goods.DeliverType {
			case DeliverCard:
				card, err := drawCard(tx, &goods)
				if err != nil {
					return err
				}
				order.DeliverContent = card
			default:
				// 文本发货（含未设置发货方式的虚拟商品）
				order.DeliverContent = goods.DeliverContent
			}
			order.Status = OrderStatusCompleted
			order.FinishTime = now
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		return nil
	})

	return order, err
}

// CancelOrder - 取消订单（用户取消自己的待发货订单，或管理员强制取消）
// 事务：校验状态 → 回滚库存与销量 → 退还积分 → 写退款流水 → 更新订单状态
func (this *GoodsOrder) CancelOrder(uid int, orderId int, isRoot bool) (order GoodsOrder, err error) {

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		if err := tx.Where("id = ?", orderId).First(&order).Error; err != nil {
			return errors.New("订单不存在！")
		}

		// 权限：管理员可取消任意订单，普通用户只能取消自己的订单
		if !isRoot && order.Uid != uid {
			return errors.New("无权操作该订单！")
		}

		// 仅「待发货」订单可取消（虚拟商品下单即完成，不涉及取消）
		if order.Status == OrderStatusCanceled {
			return errors.New("订单已取消！")
		}
		if order.Status != OrderStatusPending {
			return errors.New("当前订单状态不可取消！")
		}

		now := time.Now().Unix()

		// 1. 回滚库存
		if err := tx.Model(&Goods{}).Where("id = ?", order.GoodsId).
			UpdateColumn("stock", gorm.Expr("stock + 1")).Error; err != nil {
			return err
		}

		// 2. 销量回滚（不小于 0）
		if err := tx.Model(&Goods{}).Where("id = ? AND sold > 0", order.GoodsId).
			UpdateColumn("sold", gorm.Expr("sold - 1")).Error; err != nil {
			return err
		}

		// 3. 退还积分 + 写流水
		if order.Price > 0 {
			// 先取当前余额，用于流水快照（事务内读取，保证准确）
			var user Users
			if err := tx.Where("id = ?", order.Uid).First(&user).Error; err != nil {
				return errors.New("用户不存在！")
			}

			if err := tx.Model(&Users{}).Where("id = ?", order.Uid).
				UpdateColumn("integral", gorm.Expr("integral + ?", order.Price)).Error; err != nil {
				return err
			}

			if err := tx.Create(&Integral{
				Uid:         order.Uid,
				Value:       order.Price,
				Type:        IntegralTypeRefund,
				Description: "取消订单退款：" + order.DisplayTitle(),
				Json: utils.Json.Encode(map[string]any{
					"order_id":      order.Id,
					"order_no":      order.OrderNo,
					"goods_id":      order.GoodsId,
					"balance_after": user.Integral + order.Price,
				}),
			}).Error; err != nil {
				return err
			}
		}

		// 4. 更新订单状态
		if err := tx.Model(&GoodsOrder{}).Where("id = ?", order.Id).Updates(map[string]any{
			"status":      OrderStatusCanceled,
			"refund":      order.Price,
			"cancel_time": now,
		}).Error; err != nil {
			return err
		}

		order.Status = OrderStatusCanceled
		order.Refund = order.Price
		order.CancelTime = now

		return nil
	})

	return order, err
}

// ReceiveOrder - 确认收货（用户把自己的「已发货」订单置为「已完成」）
func (this *GoodsOrder) ReceiveOrder(uid int, orderId int) (order GoodsOrder, err error) {

	if err = facade.DB.Drive().Where("id = ?", orderId).First(&order).Error; err != nil {
		return order, errors.New("订单不存在！")
	}
	if order.Uid != uid {
		return order, errors.New("无权操作该订单！")
	}
	if order.Status != OrderStatusShipped {
		return order, errors.New("当前订单状态不可确认收货！")
	}

	now := time.Now().Unix()
	if err = facade.DB.Drive().Model(&GoodsOrder{}).Where("id = ?", orderId).
		Updates(map[string]any{
			"status":      OrderStatusCompleted,
			"finish_time": now,
		}).Error; err != nil {
		return order, err
	}

	order.Status = OrderStatusCompleted
	order.FinishTime = now
	return order, nil
}

// DisplayTitle - 订单展示用商品标题（优先快照，回退实时商品）
func (this *GoodsOrder) DisplayTitle() string {
	if !utils.Is.Empty(this.GoodsTitle) {
		return this.GoodsTitle
	}
	item, _ := facade.DB.Model(&Goods{}).Find(this.GoodsId)
	return cast.ToString(item["title"])
}

// CountGoodsBuys - 商品维度兑换统计（有效订单数 / 消耗积分）
func CountGoodsBuys(goodsId int) (orders int64, integral int) {
	var rows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS orders, COALESCE(SUM(price), 0) AS integral FROM inis_goods_order "+
			"WHERE goods_id = ? AND status != ? AND (delete_time IS NULL OR delete_time = 0)",
		goodsId, OrderStatusCanceled,
	).Scan(&rows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "goods_id": goodsId}, "商品兑换统计失败")
	}

	if len(rows) == 0 {
		return 0, 0
	}
	return cast.ToInt64(rows[0]["orders"]), cast.ToInt(rows[0]["integral"])
}

// drawCard - 从卡密池随机抽取一个卡密并移除
func drawCard(tx *gorm.DB, goods *Goods) (string, error) {
	var cards []string
	if !utils.Is.Empty(goods.Cards) {
		cards = cast.ToStringSlice(utils.Json.Decode(goods.Cards))
	}
	if len(cards) == 0 {
		return "", errors.New("卡密库存不足！")
	}

	// 随机抽取一个
	idx := utils.Rand.Int(0, len(cards)-1)
	card := cards[idx]
	// 移除该卡密
	cards = append(cards[:idx], cards[idx+1:]...)

	// 更新商品卡密池
	if err := tx.Model(&Goods{}).Where("id = ?", goods.Id).
		UpdateColumn("cards", utils.Json.Encode(cards)).Error; err != nil {
		return "", err
	}

	return card, nil
}

// result - 订单返回结果（附带商品、用户与地址信息）
func (this *GoodsOrder) result() map[string]any {
	var goods map[string]any
	var user, address any

	if this.GoodsId > 0 {
		item, _ := facade.DB.Model(&Goods{}).Find(this.GoodsId)
		if !utils.Is.Empty(item) {
			goods = cast.ToStringMap(facade.Comm.WithField(item, []any{"id", "title", "cover", "price", "type", "category"}))
		}
		if goods == nil {
			// 商品已被删除：退回下单快照
			goods = map[string]any{
				"id":    this.GoodsId,
				"title": this.GoodsTitle,
				"cover": this.GoodsCover,
				"price": this.Price,
			}
		}
		// 商品改名/改图后，订单仍展示下单时的快照
		if !utils.Is.Empty(this.GoodsTitle) {
			goods["title"] = this.GoodsTitle
		}
		if !utils.Is.Empty(this.GoodsCover) {
			goods["cover"] = this.GoodsCover
		}
	}

	if this.Uid > 0 {
		item, _ := facade.DB.Model(&Users{}).Find(this.Uid)
		if !utils.Is.Empty(item) {
			user = facade.Comm.WithField(item, []any{"id", "nickname", "avatar"})
		}
	}

	// 解析收货地址 JSON
	if !utils.Is.Empty(this.Address) {
		address = utils.Json.Decode(this.Address)
	}

	return map[string]any{"goods": goods, "user": user, "address": address}
}
