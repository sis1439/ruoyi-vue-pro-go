package migrations_test

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/product"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/trade"
	models "github.com/wxlbd/ruoyi-mall-go/internal/schema"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	"github.com/wxlbd/ruoyi-mall-go/migrations"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestPostgresMigrationCoverageAndRepeat(t *testing.T) {
	db := testutil.PostgreSQL(t)
	pool, err := db.DB()
	require.NoError(t, err)
	var version uint
	require.NoError(t, db.Raw("SELECT version FROM schema_migrations").Scan(&version).Error)
	require.Equal(t, uint(8), version)
	for _, entry := range models.Models {
		s, err := schema.Parse(entry.Model, new(sync.Map), schema.NamingStrategy{})
		require.NoError(t, err)
		var columns []string
		require.NoError(t, db.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=?", s.Table).Scan(&columns).Error)
		require.ElementsMatch(t, s.DBNames, columns, entry.Source)
		// Compile actual GORM projection against every table, including non-mall Wire dependencies.
		dest := reflect.New(reflect.SliceOf(reflect.TypeOf(entry.Model))).Interface()
		require.NoError(t, db.Limit(0).Find(dest).Error, entry.Source)
	}
	var before int64
	require.NoError(t, db.Model(&model.SystemMenu{}).Count(&before).Error)
	require.NoError(t, migrations.Apply(pool, 0))
	var after int64
	require.NoError(t, db.Model(&model.SystemMenu{}).Count(&after).Error)
	require.Equal(t, before, after)
	var pages int64
	require.NoError(t, db.Model(&model.SystemMenu{}).Where("type=2 AND component <> ?", "").Count(&pages).Error)
	require.Equal(t, int64(6), pages)
	var enabledChannels int64
	require.NoError(t, db.Model(&pay.PayChannel{}).Count(&enabledChannels).Error)
	require.Zero(t, enabledChannels)
}
func TestPostgresBaselineUpgradePreservesData(t *testing.T) {
	db := testutil.PostgreSQLAt(t, 1)
	user := member.MemberUser{Mobile: "15500000001", Nickname: "before upgrade", TenantBaseDO: model.TenantBaseDO{TenantID: 7}}
	require.NoError(t, db.Omit("Email").Create(&user).Error)
	pool, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, migrations.Apply(pool, 0))
	var got member.MemberUser
	require.NoError(t, db.First(&got, user.ID).Error)
	require.Equal(t, "before upgrade", got.Nickname)
	require.NoError(t, migrations.Apply(pool, 0))
	// Explicit seed IDs have had their sequences advanced.
	tenant := model.SystemTenant{Name: "next", Status: 0, ExpireDate: time.Now().AddDate(1, 0, 0)}
	require.NoError(t, db.Create(&tenant).Error)
	require.Greater(t, tenant.ID, int64(1))
	menu := model.SystemMenu{Name: "next", Type: 1}
	require.NoError(t, db.Create(&menu).Error)
	require.Greater(t, menu.ID, int64(15))
}
func TestPostgresTypeRoundTripsZeroUpdateAndSoftDelete(t *testing.T) {
	db := testutil.PostgreSQL(t)
	now := time.Date(2026, 9, 7, 10, 11, 12, 123456000, time.FixedZone("CST", 8*3600))
	user := member.MemberUser{Mobile: "15500000002", TagIds: model.IntListFromCSV{1, 11, 30}, LoginDate: &now, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&user).Error)
	var loaded member.MemberUser
	require.NoError(t, db.First(&loaded, user.ID).Error)
	require.Equal(t, user.TagIds, loaded.TagIds)
	require.True(t, now.Equal(*loaded.LoginDate))
	sku := product.ProductSku{SpuID: 1, Price: 3_000_000_001, Stock: 2, Properties: []product.ProductSkuProperty{{PropertyID: 1, ValueID: 2, PropertyName: "Size", ValueName: "XL"}}, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&sku).Error)
	require.False(t, bool(sku.Deleted))
	var got product.ProductSku
	require.NoError(t, db.First(&got, sku.ID).Error)
	require.Equal(t, sku.Properties, got.Properties)
	require.Equal(t, sku.Price, got.Price)
	require.NoError(t, db.Model(&sku).Updates(map[string]any{"stock": 0, "price": 0}).Error)
	require.NoError(t, db.First(&got, sku.ID).Error)
	require.Zero(t, got.Stock)
	require.Zero(t, got.Price)
	require.NoError(t, db.Delete(&sku).Error)
	require.ErrorIs(t, db.First(&got, sku.ID).Error, gorm.ErrRecordNotFound)
	require.NoError(t, db.Unscoped().First(&got, sku.ID).Error)
	require.True(t, bool(got.Deleted))
	var stored int
	require.NoError(t, db.Raw("SELECT deleted FROM product_sku WHERE id=?", sku.ID).Scan(&stored).Error)
	require.Equal(t, 1, stored)
	// Default true and explicit true both survive driver/GORM conversion.
	menu := model.SystemMenu{Name: "bool defaults", Type: 2}
	require.NoError(t, db.Create(&menu).Error)
	require.True(t, bool(menu.Visible))
	spu := product.ProductSpu{Name: "explicit true", SpecType: true, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&spu).Error)
	var found product.ProductSpu
	require.NoError(t, db.First(&found, spu.ID).Error)
	require.True(t, bool(found.SpecType))
	channel := pay.PayChannel{Code: "test", AppID: 1, Config: &pay.PayClientConfig{ConfigType: pay.ConfigTypeNone}, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&channel).Error)
	var ch pay.PayChannel
	require.NoError(t, db.First(&ch, channel.ID).Error)
	require.Equal(t, pay.ConfigTypeNone, ch.Config.ConfigType)
	pickup := trade.TradeDeliveryPickUpStore{Name: "pickup", OpeningTime: "09:30:00", ClosingTime: "18:00:00", VerifyUserIds: model.IntListFromCSV{1, 2}, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&pickup).Error)
	var p trade.TradeDeliveryPickUpStore
	require.NoError(t, db.First(&p, pickup.ID).Error)
	require.Equal(t, pickup.OpeningTime, p.OpeningTime)
}
func TestPostgresConstraintsRollbackUpsertAndLock(t *testing.T) {
	db := testutil.PostgreSQL(t)
	u := member.MemberUser{Mobile: "15500000003", TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&u).Error)
	duplicate := member.MemberUser{Mobile: u.Mobile, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.Error(t, db.Create(&duplicate).Error)
	duplicate.ID = 0
	duplicate.TenantID = 2
	require.NoError(t, db.Create(&duplicate).Error)
	require.NoError(t, db.Delete(&u).Error)
	replacement := member.MemberUser{Mobile: u.Mobile, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&replacement).Error)
	sku := product.ProductSku{SpuID: 1, Stock: 5, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}
	require.NoError(t, db.Create(&sku).Error)
	sentinel := errors.New("injected failure")
	require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error { require.NoError(t, tx.Model(&sku).Update("stock", 0).Error); return sentinel }), sentinel)
	var got product.ProductSku
	require.NoError(t, db.First(&got, sku.ID).Error)
	require.Equal(t, 5, got.Stock)
	sku.Stock = 3
	require.NoError(t, db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"stock"})}).Create(&sku).Error)
	require.NoError(t, db.First(&got, sku.ID).Error)
	require.Equal(t, 3, got.Stock)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&got, sku.ID).Error)
	competing := db.Begin()
	require.NoError(t, competing.Error)
	defer competing.Rollback()
	err := competing.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).First(&product.ProductSku{}, sku.ID).Error
	require.Error(t, err)
	require.Contains(t, err.Error(), "55P03")
}

func TestPostgresMigrationReleasesCallerConnection(t *testing.T) {
	db := testutil.PostgreSQL(t)
	pool, err := db.DB()
	require.NoError(t, err)
	require.Zero(t, pool.Stats().InUse, "migration must release its reserved connection")
	pool.SetMaxOpenConns(1)
	require.NoError(t, migrations.Apply(pool, 0))
	require.Zero(t, pool.Stats().InUse)
	require.NoError(t, pool.Ping())
}

func TestPostgresFailedUpgradeStaysDirtyAndReleasesConnection(t *testing.T) {
	db := testutil.PostgreSQLAt(t, 1)
	pool, err := db.DB()
	require.NoError(t, err)
	tenant := model.SystemTenant{ID: 1, Name: "conflicting deployment", ExpireDate: time.Now()}
	require.NoError(t, db.Create(&tenant).Error)
	pool.SetMaxOpenConns(1)
	err = migrations.Apply(pool, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "23505")
	require.Zero(t, pool.Stats().InUse)
	var state struct {
		Version int
		Dirty   bool
	}
	require.NoError(t, db.Raw("SELECT version,dirty FROM schema_migrations").Scan(&state).Error)
	require.Equal(t, 3, state.Version)
	require.True(t, state.Dirty)
	var count int64
	require.NoError(t, db.Model(&model.SystemRole{}).Count(&count).Error)
	require.Zero(t, count, "seed transaction must roll back")
	require.ErrorContains(t, migrations.Apply(pool, 0), "Dirty database")
	require.NoError(t, pool.Ping())
}

func TestPostgresTradePermissionsUpgrade(t *testing.T) {
	db := testutil.PostgreSQLAt(t, 5)
	pool, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, migrations.Apply(pool, 0))
	var source []byte
	paths, err := filepath.Glob("../internal/api/router/*.go")
	require.NoError(t, err)
	for _, path := range paths {
		if filepath.Base(path) == "iot.go" {
			continue
		}
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		source = append(source, contents...)
	}
	matches := regexp.MustCompile(`RequirePermission\("([^"\n]+)"\)`).FindAllStringSubmatch(string(source), -1)
	require.NotEmpty(t, matches)
	for _, match := range matches {
		var count int64
		require.NoError(t, db.Raw("SELECT count(*) FROM system_menu m JOIN system_role_menu rm ON rm.menu_id=m.id WHERE m.permission=? AND m.deleted=0 AND rm.deleted=0 AND rm.role_id=1 AND rm.tenant_id=1", match[1]).Scan(&count).Error)
		require.Equal(t, int64(1), count, match[1])
	}
	var count int64
	require.NoError(t, db.Model(&model.SystemRole{}).Where("code=?", "platform_admin").Count(&count).Error)
	require.Zero(t, count)
	var before int64
	require.NoError(t, db.Model(&model.SystemRoleMenu{}).Count(&before).Error)
	require.NoError(t, migrations.Apply(pool, 0))
	require.NoError(t, db.Model(&model.SystemRoleMenu{}).Count(&count).Error)
	require.Equal(t, before, count)
}

func TestJavaAlignmentUpgradeEnums(t *testing.T) {
	db := testutil.PostgreSQLAt(t, 6)
	require.NoError(t, db.Exec("INSERT INTO pay_order(id,no,status,tenant_id) VALUES (991,'old-closed',20,1),(992,'old-refund',30,1)").Error)
	for i, reason := range []int{10, 20, 50, 70} {
		require.NoError(t, db.Create(&trade.TradeOrder{ID: int64(991 + i), No: fmt.Sprint(991 + i), CancelType: reason, TenantBaseDO: model.TenantBaseDO{TenantID: 1}}).Error)
	}
	pool, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, migrations.Apply(pool, 0))
	var states []int
	require.NoError(t, db.Raw("SELECT status FROM pay_order WHERE id IN (991,992) ORDER BY id").Scan(&states).Error)
	require.Equal(t, []int{30, 20}, states)
	var reasons []int
	require.NoError(t, db.Raw("SELECT cancel_type FROM trade_order WHERE id BETWEEN 991 AND 994 ORDER BY id").Scan(&reasons).Error)
	require.Equal(t, []int{30, 10, 20, 40}, reasons)
	require.NoError(t, migrations.Apply(pool, 0))
	var again []int
	require.NoError(t, db.Raw("SELECT status FROM pay_order WHERE id IN (991,992) ORDER BY id").Scan(&again).Error)
	require.Equal(t, states, again)
	var jobs int64
	require.NoError(t, db.Raw("SELECT count(*) FROM infra_job WHERE handler_name='combinationRecordExpireJob' AND deleted=0").Scan(&jobs).Error)
	require.EqualValues(t, 1, jobs)
}
