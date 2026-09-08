package router

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterTradeRoutes 注册交易订单模块路由
func RegisterTradeRoutes(engine *gin.Engine,
	handlers *trade.Handlers,
	casbinMiddleware *middleware.CasbinMiddleware,
) {
	// Trade Order
	tradeGroup := engine.Group("/admin-api/trade/order")
	tradeGroup.Use(middleware.Auth())
	{
		tradeGroup.GET("/page", casbinMiddleware.RequirePermission("trade:order:query"), handlers.Order.GetOrderPage)
		tradeGroup.GET("/get-detail", casbinMiddleware.RequirePermission("trade:order:query"), handlers.Order.GetOrderDetail)
		tradeGroup.GET("/summary", casbinMiddleware.RequirePermission("trade:order:query"), handlers.Order.GetOrderSummary)
		tradeGroup.GET("/get-express-track-list", casbinMiddleware.RequirePermission("trade:order:query"), handlers.Order.GetOrderExpressTrackList)
		tradeGroup.GET("/get-by-pick-up-verify-code", casbinMiddleware.RequirePermission("trade:order:query"), handlers.Order.GetByPickUpVerifyCode)
		tradeGroup.PUT("/delivery", casbinMiddleware.RequirePermission("trade:order:delivery"), handlers.Order.DeliveryOrder)
		tradeGroup.PUT("/update-remark", casbinMiddleware.RequirePermission("trade:order:update-remark"), handlers.Order.UpdateOrderRemark)
		tradeGroup.PUT("/update-price", casbinMiddleware.RequirePermission("trade:order:update-price"), handlers.Order.UpdateOrderPrice)
		tradeGroup.PUT("/update-address", casbinMiddleware.RequirePermission("trade:order:update-address"), handlers.Order.UpdateOrderAddress)
		tradeGroup.PUT("/pick-up-by-id", casbinMiddleware.RequirePermission("trade:order:pick-up-by-id"), handlers.Order.PickUpOrderById)
		tradeGroup.PUT("/pick-up-by-verify-code", casbinMiddleware.RequirePermission("trade:order:pick-up-by-verify-code"), handlers.Order.PickUpOrderByVerifyCode)
	}

	// Trade AfterSale
	afterSaleGroup := engine.Group("/admin-api/trade/after-sale")
	afterSaleGroup.Use(middleware.Auth())
	{
		afterSaleGroup.GET("/page", casbinMiddleware.RequirePermission("trade:after-sale:query"), handlers.AfterSale.GetAfterSalePage)
		afterSaleGroup.GET("/get-detail", casbinMiddleware.RequirePermission("trade:after-sale:query"), handlers.AfterSale.GetAfterSaleDetail)
		afterSaleGroup.PUT("/agree", casbinMiddleware.RequirePermission("trade:after-sale:agree"), handlers.AfterSale.AgreeAfterSale)
		afterSaleGroup.PUT("/disagree", casbinMiddleware.RequirePermission("trade:after-sale:disagree"), handlers.AfterSale.DisagreeAfterSale)
		afterSaleGroup.PUT("/receive", casbinMiddleware.RequirePermission("trade:after-sale:receive"), handlers.AfterSale.ReceiveAfterSale)
		afterSaleGroup.PUT("/refuse", casbinMiddleware.RequirePermission("trade:after-sale:refuse"), handlers.AfterSale.RefuseAfterSale)
		afterSaleGroup.PUT("/refund", casbinMiddleware.RequirePermission("trade:after-sale:refund"), handlers.AfterSale.RefundAfterSale)
	}

	// Delivery Routes
	deliveryGroup := engine.Group("/admin-api/trade/delivery")
	deliveryGroup.Use(middleware.Auth())
	{
		// Express
		expressGroup := deliveryGroup.Group("/express")
		{
			expressGroup.POST("/create", casbinMiddleware.RequirePermission("trade:delivery:express:create"), handlers.DeliveryExpress.CreateDeliveryExpress)
			expressGroup.PUT("/update", casbinMiddleware.RequirePermission("trade:delivery:express:update"), handlers.DeliveryExpress.UpdateDeliveryExpress)
			expressGroup.DELETE("/delete", casbinMiddleware.RequirePermission("trade:delivery:express:delete"), handlers.DeliveryExpress.DeleteDeliveryExpress)
			expressGroup.GET("/get", casbinMiddleware.RequirePermission("trade:delivery:express:query"), handlers.DeliveryExpress.GetDeliveryExpress)
			expressGroup.GET("/page", casbinMiddleware.RequirePermission("trade:delivery:express:query"), handlers.DeliveryExpress.GetDeliveryExpressPage)
			expressGroup.GET("/list-all-simple", casbinMiddleware.RequirePermission("trade:delivery:express:query"), handlers.DeliveryExpress.GetSimpleDeliveryExpressList)
			expressGroup.GET("/export-excel", casbinMiddleware.RequirePermission("trade:delivery:express:query"), handlers.DeliveryExpress.ExportDeliveryExpress)
		}

		// Pick Up Store
		pickUpStoreGroup := deliveryGroup.Group("/pick-up-store")
		{
			pickUpStoreGroup.POST("/create", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:create"), handlers.DeliveryPickUpStore.CreateDeliveryPickUpStore)
			pickUpStoreGroup.PUT("/update", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:update"), handlers.DeliveryPickUpStore.UpdateDeliveryPickUpStore)
			pickUpStoreGroup.DELETE("/delete", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:delete"), handlers.DeliveryPickUpStore.DeleteDeliveryPickUpStore)
			pickUpStoreGroup.GET("/get", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:query"), handlers.DeliveryPickUpStore.GetDeliveryPickUpStore)
			pickUpStoreGroup.GET("/page", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:query"), handlers.DeliveryPickUpStore.GetDeliveryPickUpStorePage)
			pickUpStoreGroup.GET("/simple-list", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:query"), handlers.DeliveryPickUpStore.GetSimpleDeliveryPickUpStoreList)
			pickUpStoreGroup.POST("/bind", casbinMiddleware.RequirePermission("trade:delivery:pick-up-store:bind"), handlers.DeliveryPickUpStore.BindDeliveryPickUpStore)
		}

		// Express Template (运费模板) - 对齐 Java 路径
		expressTemplateGroup := deliveryGroup.Group("/express-template")
		{
			expressTemplateGroup.POST("/create", casbinMiddleware.RequirePermission("trade:delivery:express-template:create"), handlers.DeliveryExpressTemplate.CreateDeliveryExpressTemplate)
			expressTemplateGroup.PUT("/update", casbinMiddleware.RequirePermission("trade:delivery:express-template:update"), handlers.DeliveryExpressTemplate.UpdateDeliveryExpressTemplate)
			expressTemplateGroup.DELETE("/delete", casbinMiddleware.RequirePermission("trade:delivery:express-template:delete"), handlers.DeliveryExpressTemplate.DeleteDeliveryExpressTemplate)
			expressTemplateGroup.GET("/get", casbinMiddleware.RequirePermission("trade:delivery:express-template:query"), handlers.DeliveryExpressTemplate.GetDeliveryExpressTemplate)
			expressTemplateGroup.GET("/page", casbinMiddleware.RequirePermission("trade:delivery:express-template:query"), handlers.DeliveryExpressTemplate.GetDeliveryExpressTemplatePage)
			expressTemplateGroup.GET("/list-all-simple", casbinMiddleware.RequirePermission("trade:delivery:express-template:query"), handlers.DeliveryExpressTemplate.GetSimpleDeliveryExpressTemplateList)
		}
	}

	// Trade Config (Admin)
	tradeConfigGroup := engine.Group("/admin-api/trade/config")
	tradeConfigGroup.Use(middleware.Auth())
	{
		tradeConfigGroup.GET("/get", casbinMiddleware.RequirePermission("trade:config:query"), handlers.Config.GetTradeConfig)
		tradeConfigGroup.PUT("/save", casbinMiddleware.RequirePermission("trade:config:save"), handlers.Config.SaveTradeConfig)
	}

	// Brokerage User
	brokerageUserGroup := engine.Group("/admin-api/trade/brokerage-user")
	brokerageUserGroup.Use(middleware.Auth())
	{
		brokerageUserGroup.POST("/create", casbinMiddleware.RequirePermission("trade:brokerage-user:create"), handlers.Brokerage.BrokerageUser.CreateBrokerageUser)
		brokerageUserGroup.PUT("/update-bind-user", casbinMiddleware.RequirePermission("trade:brokerage-user:update-bind-user"), handlers.Brokerage.BrokerageUser.UpdateBindUser)
		brokerageUserGroup.PUT("/clear-bind-user", casbinMiddleware.RequirePermission("trade:brokerage-user:clear-bind-user"), handlers.Brokerage.BrokerageUser.ClearBindUser)
		brokerageUserGroup.PUT("/update-brokerage-enable", casbinMiddleware.RequirePermission("trade:brokerage-user:update-brokerage-enable"), handlers.Brokerage.BrokerageUser.UpdateBrokerageEnabled)
		brokerageUserGroup.GET("/get", casbinMiddleware.RequirePermission("trade:brokerage-user:query"), handlers.Brokerage.BrokerageUser.GetBrokerageUser)
		brokerageUserGroup.GET("/page", casbinMiddleware.RequirePermission("trade:brokerage-user:query"), handlers.Brokerage.BrokerageUser.GetBrokerageUserPage)
	}

	// Brokerage Record
	brokerageRecordGroup := engine.Group("/admin-api/trade/brokerage-record")
	brokerageRecordGroup.Use(middleware.Auth())
	{
		brokerageRecordGroup.GET("/get", casbinMiddleware.RequirePermission("trade:brokerage-record:query"), handlers.Brokerage.BrokerageRecord.GetBrokerageRecord)
		brokerageRecordGroup.GET("/page", casbinMiddleware.RequirePermission("trade:brokerage-record:query"), handlers.Brokerage.BrokerageRecord.GetBrokerageRecordPage)
	}

	// Brokerage Withdraw
	brokerageWithdrawGroup := engine.Group("/admin-api/trade/brokerage-withdraw")
	brokerageWithdrawGroup.Use(middleware.Auth())
	{
		brokerageWithdrawGroup.PUT("/approve", casbinMiddleware.RequirePermission("trade:brokerage-withdraw:approve"), handlers.Brokerage.BrokerageWithdraw.ApproveBrokerageWithdraw)
		brokerageWithdrawGroup.PUT("/reject", casbinMiddleware.RequirePermission("trade:brokerage-withdraw:reject"), handlers.Brokerage.BrokerageWithdraw.RejectBrokerageWithdraw)
		brokerageWithdrawGroup.POST("/update-transferred", casbinMiddleware.RequirePermission("trade:brokerage-withdraw:update-transferred"), handlers.Brokerage.BrokerageWithdraw.UpdateBrokerageWithdrawTransferred)
		brokerageWithdrawGroup.GET("/get", casbinMiddleware.RequirePermission("trade:brokerage-withdraw:query"), handlers.Brokerage.BrokerageWithdraw.GetBrokerageWithdraw)
		brokerageWithdrawGroup.GET("/page", casbinMiddleware.RequirePermission("trade:brokerage-withdraw:query"), handlers.Brokerage.BrokerageWithdraw.GetBrokerageWithdrawPage)
	}

	// Trade AfterSale Callback：支付中心 → 商城的退款业务通知，
	// 无用户身份，改由内部调用令牌建立信任边界（服务端仍重新校验退款单）
	afterSaleCallbackGroup := engine.Group("/admin-api/trade/after-sale")
	afterSaleCallbackGroup.Use(middleware.PayNotifyToken())
	{
		afterSaleCallbackGroup.POST("/update-refunded", handlers.AfterSale.UpdateAfterSaleRefunded)
	}
}
