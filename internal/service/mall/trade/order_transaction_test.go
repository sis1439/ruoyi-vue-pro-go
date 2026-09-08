package trade

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tradeapi "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	membermodel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	paymodel "github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	productmodel "github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	promotionmodel "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	trademodel "github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	productservice "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/product"
	promotionservice "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	memberservice "github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	payservice "github.com/wxlbd/ruoyi-mall-go/internal/service/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenantcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The real pricing service still loads SKU/SPU rows. Only the discount calculation
// is fixed so the tests isolate resource consistency, not marketing price formulas.
type transactionDiscount struct{ *BasePriceCalculator }

func (transactionDiscount) Calculate(_ context.Context, req *TradePriceCalculateReqBO, resp *TradePriceCalculateRespBO) error {
	if req.CouponID != nil && *req.CouponID > 0 {
		resp.CouponID = *req.CouponID
		resp.Items[0].CouponPrice = 10
		resp.Items[0].PayPrice -= 10
	}
	if req.PointStatus {
		resp.UsePoint = 10
		resp.Items[0].UsePoint = 10
		resp.Items[0].PointPrice = 10
		resp.Items[0].PayPrice -= 10
	}
	return nil
}
func (transactionDiscount) GetName() string       { return "transaction-test-discount" }
func (transactionDiscount) GetOrder() int         { return 1 }
func (transactionDiscount) IsApplicable(int) bool { return true }

type transactionBefore struct {
	*BaseOrderHandler
	run func() error
}

func (h *transactionBefore) BeforeOrderCreate(context.Context, *OrderHandleRequest) error {
	return h.run()
}

type orderFixture struct {
	db      *gorm.DB
	ctx     context.Context
	svc     *TradeOrderUpdateService
	initial []int
	points  int32
}

func newOrderFixture(t *testing.T, stock []int, points int32) *orderFixture {
	t.Helper()
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	ctx, svc := transactionServices(db)
	total := 0
	for _, n := range stock {
		total += n
	}
	require.NoError(t, db.WithContext(ctx).Create(&productmodel.ProductSpu{ID: 1, Name: "test", Stock: total, Status: consts.ProductSpuStatusEnable}).Error)
	for i, n := range stock {
		require.NoError(t, db.WithContext(ctx).Create(&productmodel.ProductSku{ID: int64(i + 1), SpuID: 1, Price: 100, Stock: n}).Error)
		require.NoError(t, db.WithContext(ctx).Create(&trademodel.Cart{ID: int64(i + 1), UserID: 1, SpuID: 1, SkuID: int64(i + 1), Count: 1, Selected: true}).Error)
	}
	require.NoError(t, db.WithContext(ctx).Create(&membermodel.MemberUser{ID: 1, Mobile: "13800000000", Point: points}).Error)
	require.NoError(t, db.WithContext(ctx).Create(&promotionmodel.PromotionCoupon{ID: 1, Name: "test", UserID: 1, Status: consts.CouponStatusUnused, ValidStartTime: time.Now().Add(-time.Hour), ValidEndTime: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.WithContext(ctx).Model(&paymodel.PayApp{}).Where("app_key = ?", "mall").Update("status", 0).Error)
	return &orderFixture{db: db, ctx: ctx, svc: svc, initial: stock, points: points}
}
func (f *orderFixture) request(coupon, points bool, ids ...int64) *tradeapi.AppTradeOrderCreateReq {
	r := &tradeapi.AppTradeOrderCreateReq{AppTradeOrderSettlementReq: tradeapi.AppTradeOrderSettlementReq{DeliveryType: consts.DeliveryTypePickUp, PointStatus: &points}}
	if coupon {
		id := int64(1)
		r.CouponID = &id
	}
	for _, id := range ids {
		r.Items = append(r.Items, tradeapi.AppTradeOrderSettlementItem{SkuID: id, CartID: id, Count: 1})
	}
	return r
}
func (f *orderFixture) before(t *testing.T, fn func() error) {
	t.Helper()
	require.NoError(t, f.svc.InitializeHandlers([]OrderHandler{&transactionBefore{BaseOrderHandler: NewBaseOrderHandler("test", nil), run: fn}}))
}
func (f *orderFixture) create(r *tradeapi.AppTradeOrderCreateReq) (*trademodel.TradeOrder, error) {
	return f.svc.CreateOrder(f.ctx, 1, "127.0.0.1", 10, r)
}

// Assert every durable resource, including the local pay linkage and absence of remote work.
func (f *orderFixture) assertState(t *testing.T, stock []int, orderCount int, usedCoupon bool, points int32, ledgerPoints []int) {
	t.Helper()
	db := f.db.WithContext(f.ctx)
	var skus []productmodel.ProductSku
	require.NoError(t, db.Order("id").Find(&skus).Error)
	require.Len(t, skus, len(stock))

	totals := map[int64]int{}
	for i, n := range stock {
		require.Equal(t, n, skus[i].Stock)
		require.Equal(t, f.initial[i]-n, skus[i].SalesCount)
		totals[skus[i].SpuID] += n
	}
	var spus []productmodel.ProductSpu
	require.NoError(t, db.Find(&spus).Error)
	require.Len(t, spus, len(totals))
	for _, spu := range spus {
		require.Equal(t, totals[spu.ID], spu.Stock)
	}

	var coupon promotionmodel.PromotionCoupon
	require.NoError(t, db.First(&coupon, 1).Error)
	if usedCoupon {
		require.Equal(t, consts.CouponStatusUsed, coupon.Status)
		require.NotZero(t, coupon.UseOrderID)
		require.NotNil(t, coupon.UseTime)
	} else {
		require.Equal(t, consts.CouponStatusUnused, coupon.Status)
		require.Zero(t, coupon.UseOrderID)
		require.Nil(t, coupon.UseTime)
	}
	var user membermodel.MemberUser
	require.NoError(t, db.First(&user, 1).Error)
	require.Equal(t, points, user.Point)
	var ledger []membermodel.MemberPointRecord
	require.NoError(t, db.Order("id").Find(&ledger).Error)
	require.Len(t, ledger, len(ledgerPoints))
	balance := int(f.points)
	for i, n := range ledgerPoints {
		balance += n
		require.Equal(t, n, ledger[i].Point)
		require.Equal(t, balance, ledger[i].TotalPoint)
		require.NotEmpty(t, ledger[i].BizID)
	}
	var orders []trademodel.TradeOrder
	require.NoError(t, db.Find(&orders).Error)
	require.Len(t, orders, orderCount)
	var pays []paymodel.PayOrder
	require.NoError(t, db.Find(&pays).Error)
	require.Len(t, pays, orderCount)
	var items []trademodel.TradeOrderItem
	require.NoError(t, db.Find(&items).Error)
	expectedItems := 0
	for _, o := range orders {
		require.NotNil(t, o.PayOrderID)
		require.False(t, bool(o.PayStatus))
		require.Positive(t, o.PayPrice)
		matched := false
		for _, p := range pays {
			if p.ID == *o.PayOrderID {
				matched = true
				require.Equal(t, strconv.FormatInt(o.ID, 10), p.MerchantOrderId)
				require.Equal(t, o.PayPrice, p.Price)
				require.Equal(t, consts.PayOrderStatusWaiting, p.Status)
			}
		}
		require.True(t, matched)
		count := 0
		for _, item := range items {
			if item.OrderID == o.ID {
				count += item.Count
				expectedItems++
				require.Equal(t, o.UserID, item.UserID)
			}
		}
		require.Equal(t, o.ProductCount, count)
		if usedCoupon {
			require.Equal(t, o.ID, coupon.UseOrderID)
		}
		for _, l := range ledger {
			require.Equal(t, fmt.Sprint(o.ID), l.BizID)
		}
	}
	require.Equal(t, expectedItems, len(items))
	var carts []trademodel.Cart
	require.NoError(t, db.Unscoped().Order("id").Find(&carts).Error)
	require.Len(t, carts, len(f.initial))
	usedCarts := map[int64]bool{}
	for _, item := range items {
		usedCarts[item.CartID] = true
	}
	for _, cart := range carts {
		require.Equal(t, usedCarts[cart.ID], bool(cart.Deleted))
		require.Equal(t, 1, cart.Count)
	}
	var logs []trademodel.TradeOrderLog
	require.NoError(t, db.Find(&logs).Error)
	expectedLogs := orderCount
	for _, order := range orders {
		if order.Status == consts.TradeOrderStatusCanceled {
			expectedLogs++
		}
	}
	require.Len(t, logs, expectedLogs)
	for _, log := range logs {
		found := false
		for _, order := range orders {
			if order.ID == log.OrderID {
				found = true
			}
		}
		require.True(t, found)
	}

	for _, model := range []any{&paymodel.PayOrderExtension{}, &paymodel.PayNotifyTask{}, &paymodel.PayRefund{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestOrderTransactionPostgresFailures(t *testing.T) {
	for _, stage := range []string{"second-sku", "coupon", "points", "ledger", "pay", "pay-link", "cart", "log", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newOrderFixture(t, []int{2, 2}, 10)
			if stage == "second-sku" {
				f.before(t, func() error {
					if err := f.db.WithContext(f.ctx).Model(&productmodel.ProductSku{}).Where("id = ?", 2).Update("stock", 0).Error; err != nil {
						return err
					}
					return f.db.WithContext(f.ctx).Model(&productmodel.ProductSpu{}).Where("id = ?", 1).Update("stock", 2).Error
				})
				f.initial[1] = 0
			}
			if stage == "coupon" {
				require.NoError(t, f.db.WithContext(f.ctx).Model(&promotionmodel.PromotionCoupon{}).Where("id = ?", 1).Update("valid_end_time", time.Now().Add(-time.Second)).Error)
			}
			if stage == "points" {
				require.NoError(t, f.db.WithContext(f.ctx).Model(&membermodel.MemberUser{}).Where("id = ?", 1).Update("point", 9).Error)
				f.points = 9
			}
			table := ""
			switch stage {
			case "ledger":
				table = "member_point_record"
			case "pay":
				table = "pay_order"
			case "log":
				table = "trade_order_log"
			}
			if table != "" {
				require.NoError(t, f.db.Callback().Create().Before("gorm:create").Register("test:fail", func(db *gorm.DB) {
					if db.Statement.Table == table {
						db.AddError(errors.New("injected " + stage))
					}
				}))
				defer f.db.Callback().Create().Remove("test:fail")
			}
			if stage == "pay-link" {
				require.NoError(t, f.db.Callback().Update().Before("gorm:update").Register("test:fail", func(db *gorm.DB) {
					if db.Statement.Table == "trade_order" {
						db.AddError(errors.New("injected pay-link"))
					}
				}))
				defer f.db.Callback().Update().Remove("test:fail")
			}
			if stage == "cart" {
				require.NoError(t, f.db.Callback().Delete().Before("gorm:delete").Register("test:fail", func(db *gorm.DB) {
					if db.Statement.Table == "trade_cart" {
						db.AddError(errors.New("injected cart"))
					}
				}))
				defer f.db.Callback().Delete().Remove("test:fail")
			}

			if stage == "commit" {
				// Deferred trigger fires only at COMMIT, after all SQL writes succeeded.
				pool, e := f.db.DB()
				require.NoError(t, e)
				_, e = pool.Exec(`CREATE FUNCTION reject_order_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected commit failure'; END $$`)
				require.NoError(t, e)
				_, e = pool.Exec(`CREATE CONSTRAINT TRIGGER reject_order_commit AFTER INSERT ON trade_order DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_order_commit()`)
				require.NoError(t, e)
			}
			order, err := f.create(f.request(true, true, 1, 2))
			require.Error(t, err)
			switch stage {
			case "ledger", "pay", "pay-link", "cart", "log":
				require.ErrorContains(t, err, "injected "+stage)
			case "commit":
				require.ErrorContains(t, err, "injected commit failure")
			}
			require.Nil(t, order)
			f.assertState(t, f.initial, 0, false, f.points, nil)
		})
	}
}

func TestOrderTransactionPostgresCompetition(t *testing.T) {
	for _, resource := range []string{"last-stock", "coupon", "points"} {
		t.Run(resource, func(t *testing.T) {
			stocks := []int{1}
			if resource != "last-stock" {
				stocks = []int{1, 1}
			}
			f := newOrderFixture(t, stocks, 10)
			ids := []int64{1, 1}
			if resource != "last-stock" {
				// Different SPUs avoid serializing coupon/point contenders on a shared stock row.
				require.NoError(t, f.db.WithContext(f.ctx).Model(&productmodel.ProductSpu{}).Where("id = ?", 1).Update("stock", 1).Error)
				require.NoError(t, f.db.WithContext(f.ctx).Create(&productmodel.ProductSpu{ID: 2, Name: "second", Stock: 1, Status: consts.ProductSpuStatusEnable}).Error)
				require.NoError(t, f.db.WithContext(f.ctx).Model(&productmodel.ProductSku{}).Where("id = ?", 2).Update("spu_id", 2).Error)
				ids[1] = 2
			}
			var barrier sync.WaitGroup
			barrier.Add(2)
			f.before(t, func() error { barrier.Done(); barrier.Wait(); return nil })
			type result struct {
				id  int64
				err error
			}
			results := make(chan result, 2)
			for _, id := range ids {
				go func(id int64) {
					_, err := f.create(f.request(resource == "coupon", resource == "points", id))
					results <- result{id, err}
				}(id)
			}
			succeeded := 0
			remaining := append([]int(nil), stocks...)
			for range 2 {
				select {
				case r := <-results:
					if r.err == nil {
						succeeded++
						remaining[r.id-1]--
					}
				case <-time.After(15 * time.Second):
					t.Fatal("concurrent order stalled")
				}
			}
			require.Equal(t, 1, succeeded)
			remainingPoints := int32(10)
			var ledger []int
			if resource == "points" {
				remainingPoints = 0
				ledger = []int{-10}
			}
			f.assertState(t, remaining, 1, resource == "coupon", remainingPoints, ledger)
		})
	}
}

func TestOrderTransactionPostgresCancel(t *testing.T) {
	f := newOrderFixture(t, []int{2, 2}, 10)
	o, err := f.create(f.request(true, true, 1, 2))
	require.NoError(t, err)
	f.assertState(t, []int{1, 1}, 1, true, 0, []int{-10})

	var saved trademodel.TradeOrder
	for _, table := range []string{"member_point_record", "trade_order_log"} {
		require.NoError(t, f.db.Callback().Create().Before("gorm:create").Register("test:cancel-fail", func(db *gorm.DB) {
			if db.Statement.Table == table {
				db.AddError(errors.New("injected return failure"))
			}
		}))
		require.ErrorContains(t, f.svc.CancelOrder(f.ctx, 1, o.ID), "injected return failure")
		require.NoError(t, f.db.Callback().Create().Remove("test:cancel-fail"))
		f.assertState(t, []int{1, 1}, 1, true, 0, []int{-10})
		require.NoError(t, f.db.WithContext(f.ctx).First(&saved, o.ID).Error)
		require.Equal(t, consts.TradeOrderStatusUnpaid, saved.Status)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- f.svc.CancelOrder(f.ctx, 1, o.ID) }()
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	require.Equal(t, 1, success)
	require.Error(t, f.svc.cancelOrderBySystemSingle(f.ctx, o))
	f.assertState(t, []int{2, 2}, 1, false, 10, []int{-10, 10})
	require.NoError(t, f.db.WithContext(f.ctx).First(&saved, o.ID).Error)
	require.Equal(t, consts.TradeOrderStatusCanceled, saved.Status)
}

func TestOrderTransactionRejectsUnsupportedInput(t *testing.T) {
	svc := &TradeOrderUpdateService{}
	_, err := svc.CreateOrder(context.Background(), 1, "", 0, nil)
	require.Error(t, err)
	r := &tradeapi.AppTradeOrderCreateReq{AppTradeOrderSettlementReq: tradeapi.AppTradeOrderSettlementReq{Items: []tradeapi.AppTradeOrderSettlementItem{{SkuID: 1, Count: 0}}}}
	_, err = svc.CreateOrder(context.Background(), 1, "", 0, r)
	require.Error(t, err)
	r.Items[0].Count = 1
	id := int64(1)
	r.SeckillActivityID = &id
	_, err = svc.CreateOrder(context.Background(), 1, "", 0, r)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "营销活动"))
}

func (h *transactionBefore) Handle(context.Context, *OrderHandleRequest) (*OrderHandleResponse, error) {
	return &OrderHandleResponse{Success: true}, nil
}

func transactionServices(db *gorm.DB) (context.Context, *TradeOrderUpdateService) {
	ctx := tenantcontext.WithTenant(context.Background(), 1)
	q := query.Use(db)
	value := productservice.NewProductPropertyValueService(q)
	sku := productservice.NewProductSkuService(q, productservice.NewProductPropertyService(q, value), value)
	spu := productservice.NewProductSpuService(q, sku, nil, nil)
	member := memberservice.NewMemberUserService(q, nil, nil, nil)
	coupon := promotionservice.NewCouponUserService(q)
	app := payservice.NewPayAppService(q, nil)
	// No channel factory, notify service, or network client is supplied: local pay rows only.
	pay := payservice.NewPayOrderService(q, app, nil, nil, nil, nil)
	price := &TradePriceService{skuSvc: sku, spuSvc: spu, helper: NewPriceCalculatorHelper(zap.NewNop()), calculators: []PriceCalculator{transactionDiscount{}}, logger: zap.NewNop()}
	svc := NewTradeOrderUpdateService(q, price, NewCartService(q, sku, spu), nil, pay, nil, app, NewTradeConfigService(q), sku, nil, coupon, member, nil, nil, zap.NewNop())
	return ctx, svc
}

// A subprocess is killed after local pay INSERT, before linking/commit. No Go
// defers run; PostgreSQL must roll back when the process drops its connection.
func TestOrderTransactionPostgresProcessExit(t *testing.T) {
	if dsn := os.Getenv("T04_CRASH_DSN"); dsn != "" {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		require.NoError(t, err)
		require.NoError(t, db.Use(&database.TenantPlugin{}))
		ctx, svc := transactionServices(db)
		require.NoError(t, db.Callback().Create().After("gorm:create").Register("test:kill", func(tx *gorm.DB) {
			if tx.Statement.Table == "pay_order" && tx.Error == nil {
				p, _ := os.FindProcess(os.Getpid())
				_ = p.Kill()
				select {}
			}
		}))
		f := &orderFixture{ctx: ctx, svc: svc}
		_, err = f.create(f.request(true, true, 1, 2))
		t.Fatalf("child did not reach kill point: %v", err)
	}
	f := newOrderFixture(t, []int{2, 2}, 10)
	pool, err := f.db.DB()
	require.NoError(t, err)
	var schema string
	require.NoError(t, pool.QueryRow("SHOW search_path").Scan(&schema))
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	// This test uses the key/value PostgreSQL DSN documented in the local test command.
	if strings.Contains(dsn, "://") {
		t.Skip("process interruption test requires key/value DSN")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestOrderTransactionPostgresProcessExit$", "-test.timeout=15s")
	cmd.Env = append(os.Environ(), "T04_CRASH_DSN="+dsn+" search_path="+schema)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))
	require.Contains(t, err.Error(), "signal: killed", string(output))
	require.Eventually(t, func() bool {
		var n int64
		return f.db.WithContext(f.ctx).Model(&trademodel.TradeOrder{}).Count(&n).Error == nil && n == 0
	}, time.Second*3, time.Millisecond*10)
	f.assertState(t, []int{2, 2}, 0, false, 10, nil)
}

func TestOrderTransactionPostgresZeroPrice(t *testing.T) {
	f := newOrderFixture(t, []int{1}, 10)
	require.NoError(t, f.db.WithContext(f.ctx).Model(&productmodel.ProductSku{}).Where("id = ?", 1).Update("price", 0).Error)
	order, err := f.create(f.request(false, false, 1))
	require.Error(t, err)
	require.Nil(t, order)
	f.assertState(t, []int{1}, 0, false, 10, nil)
}

func TestOrderTransactionPostgresMemberTagCount(t *testing.T) {
	f := newOrderFixture(t, []int{1}, 10)
	require.NoError(t, f.db.WithContext(f.ctx).Model(&membermodel.MemberUser{}).Where("id = ?", 1).Update("tag_ids", "1,11").Error)
	svc := memberservice.NewMemberUserService(query.Use(f.db), nil, nil, nil)
	for _, id := range []int64{1, 11, 111} {
		n, err := svc.GetUserCountByTagId(f.ctx, id)
		require.NoError(t, err)
		expected := int64(1)
		if id == 111 {
			expected = 0
		}
		require.Equal(t, expected, n)
	}
}

func TestOrderTransactionPostgresSingleConnection(t *testing.T) {
	f := newOrderFixture(t, []int{1}, 10)
	pool, err := f.db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	f.ctx = ctx
	_, err = f.create(f.request(true, true, 1))
	require.NoError(t, err)
	f.assertState(t, []int{0}, 1, true, 0, []int{-10})
}

// Uses the actual price service SKU/SPU validation before discount calculation.
func TestOrderTransactionUnavailableProduct(t *testing.T) {
	for _, unavailable := range []string{"out-of-stock", "off-shelf", "recycled"} {
		t.Run(unavailable, func(t *testing.T) {
			stock := 1
			if unavailable == "out-of-stock" {
				stock = 0
			}
			f := newOrderFixture(t, []int{stock}, 10)
			if unavailable != "out-of-stock" {
				status := consts.ProductSpuStatusDisable
				if unavailable == "recycled" {
					status = consts.ProductSpuStatusRecycle
				}
				require.NoError(t, f.db.WithContext(f.ctx).Model(&productmodel.ProductSpu{}).Where("id = ?", 1).Update("status", status).Error)
			}
			order, err := f.create(f.request(true, true, 1))
			require.Error(t, err)
			require.Nil(t, order)
			f.assertState(t, []int{stock}, 0, false, 10, nil)
		})
	}
}
