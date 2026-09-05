package model

import (
	"errors"
	"inis/app/facade"

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
	Id             int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Title          string `gorm:"size:128; comment:商品名称;" json:"title"`
	Description    string `gorm:"type:text; comment:商品描述; default:Null;" json:"description"`
	Cover          string `gorm:"comment:商品封面; default:Null;" json:"cover"`
	Price          int    `gorm:"type:int(32); comment:积分价格; default:0;" json:"price"`
	Stock          int    `gorm:"type:int(32); comment:库存; default:0;" json:"stock"`
	Status         int    `gorm:"tinyint; default:1; comment:状态（0下架 1上架）;" json:"status"`
	Type           string `gorm:"size:16; comment:商品类型（virtual虚拟 physical实物）; default:'virtual';" json:"type"`
	DeliverType    string `gorm:"size:16; comment:发货方式（text文本 card卡密，仅虚拟商品）; default:'';" json:"deliver_type"`
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
	Uid            int    `gorm:"type:int(32); index; comment:用户ID;" json:"uid"`
	GoodsId        int    `gorm:"type:int(32); comment:商品ID;" json:"goods_id"`
	Price          int    `gorm:"type:int(32); comment:成交积分价格; default:0;" json:"price"`
	Status         int    `gorm:"tinyint; default:0; comment:状态（0待发货 1已发货 2已完成）;" json:"status"`
	DeliverContent string `gorm:"type:text; comment:发货内容（虚拟商品：文本/卡密）; default:Null;" json:"deliver_content"`
	Address        string `gorm:"type:text; comment:收货地址（JSON，实物商品）; default:Null;" json:"address"`
	Logistics      string `gorm:"type:text; comment:物流信息（实物商品发货）; default:Null;" json:"logistics"`
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

// Buy - 购买商品（事务：校验库存 → 扣库存 → 扣积分 → 写流水 → 生成订单 → 虚拟商品立即发货）
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

		// 2. 实物商品校验收货地址
		if goods.Type == GoodsTypePhysical && utils.Is.Empty(address) {
			return errors.New("请填写收货地址！")
		}

		// 3. 卡密商品校验卡密池
		if goods.Type == GoodsTypeVirtual && goods.DeliverType == DeliverCard && utils.Is.Empty(goods.Cards) {
			return errors.New("卡密库存不足！")
		}

		// 4. 校验并扣减库存（原子操作，防止超卖）
		result := tx.Model(&Goods{}).Where("id = ? AND stock > 0", goodsId).
			UpdateColumn("stock", gorm.Expr("stock - 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("商品库存不足！")
		}

		// 5. 校验积分余额并扣减
		var user Users
		if err := tx.Where("id = ?", uid).First(&user).Error; err != nil {
			return errors.New("用户不存在！")
		}
		if user.Integral < goods.Price {
			return errors.New("积分不足！")
		}

		if err := tx.Model(&Users{}).Where("id = ?", uid).
			UpdateColumn("integral", gorm.Expr("integral - ?", goods.Price)).Error; err != nil {
			return err
		}

		// 6. 写入积分流水
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       -goods.Price,
			Type:        "buy",
			Description: "购买商品：" + goods.Title,
		}).Error; err != nil {
			return err
		}

		// 7. 生成订单
		order = GoodsOrder{
			Uid:     uid,
			GoodsId: goodsId,
			Price:   goods.Price,
			Status:  OrderStatusPending,
		}

		// 8. 根据商品类型处理发货
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
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		return nil
	})

	return order, err
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
	var goods, user, address any
	if this.GoodsId > 0 {
		item, _ := facade.DB.Model(&Goods{}).Find(this.GoodsId)
		if !utils.Is.Empty(item) {
			goods = facade.Comm.WithField(item, []any{"id", "title", "cover", "price", "type"})
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
