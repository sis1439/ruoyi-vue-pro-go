package router

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app"
	"github.com/wxlbd/ruoyi-mall-go/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAppRoutes 注册 App 端路由
func RegisterAppRoutes(engine *gin.Engine,
	handlers *app.AppHandlers,
) {
	appGroup := engine.Group("/app-api")
	{
		// 文件写入沿用租户和登录身份校验。
		appGroup.POST("/infra/file/upload", middleware.Auth(), handlers.Support.UploadFile)
		appGroup.POST("/infra/file/create", middleware.Auth(), handlers.Support.CreateFile)
		appGroup.GET("/infra/file/presigned-url", middleware.Auth(), handlers.Support.GetFilePresignedUrl)
		appGroup.GET("/system/area/tree", handlers.Support.GetAreaTree)
		appGroup.GET("/system/dict-data/type", handlers.Support.GetDictData)
		appGroup.GET("/trade/delivery/express/list", handlers.Support.GetExpressList)
		appGroup.GET("/trade/delivery/pick-up-store/list", handlers.Support.GetPickUpStoreList)
		appGroup.GET("/trade/delivery/pick-up-store/get", handlers.Support.GetPickUpStore)
		appGroup.GET("/trade/after-sale-log/list", middleware.Auth(), handlers.Support.GetAfterSaleLogs)
		// ========== System ==========
		systemGroup := appGroup.Group("/system")
		{
			// Tenant (Public - 对齐 Java @PermitAll)
			systemGroup.GET("/tenant/get-by-website", handlers.System.Tenant.GetTenantByWebsite)
		}

		// ========== Member ==========
		memberGroup := appGroup.Group("/member")
		{
			// Auth
			authGroup := memberGroup.Group("/auth")
			{
				authGroup.POST("/login", handlers.Member.Auth.Login)
				authGroup.POST("/sms-login", handlers.Member.Auth.SmsLogin)
				authGroup.POST("/social-login", handlers.Member.Auth.SocialLogin)
				authGroup.POST("/send-sms-code", middleware.OptionalAuth(), handlers.Member.Auth.SendSmsCode)
				authGroup.POST("/validate-sms-code", handlers.Member.Auth.ValidateSmsCode)
				authGroup.POST("/logout", handlers.Member.Auth.Logout)
				authGroup.POST("/refresh-token", handlers.Member.Auth.RefreshToken)
				authGroup.GET("/social-auth-redirect", handlers.Member.Auth.SocialAuthRedirect)
				authGroup.POST("/weixin-mini-app-login", handlers.Member.Auth.WeixinMiniAppLogin)
				authGroup.POST("/create-weixin-jsapi-signature", handlers.Member.Auth.CreateWeixinMpJsapiSignature)
			}

			// User (Auth Required)
			userGroup := memberGroup.Group("/user")
			userGroup.Use(middleware.Auth())
			{
				userGroup.GET("/get", handlers.Member.User.GetUserInfo)
				userGroup.PUT("/update", handlers.Member.User.UpdateUser)
				userGroup.PUT("/update-mobile", handlers.Member.User.UpdateUserMobile)
				userGroup.PUT("/update-password", handlers.Member.User.UpdateUserPassword)
				userGroup.PUT("/update-mobile-by-weixin", handlers.Member.User.UpdateUserMobileByWeixin)
			}

			// User (Public)
			userPublicGroup := memberGroup.Group("/user")
			{
				userPublicGroup.PUT("/reset-password", handlers.Member.User.ResetUserPassword)
			}

			// Address (Auth Required)
			addressGroup := memberGroup.Group("/address")
			addressGroup.Use(middleware.Auth())
			{
				addressGroup.POST("/create", handlers.Member.Address.CreateAddress)
				addressGroup.PUT("/update", handlers.Member.Address.UpdateAddress)
				addressGroup.DELETE("/delete", handlers.Member.Address.DeleteAddress)
				addressGroup.GET("/get", handlers.Member.Address.GetAddress)
				addressGroup.GET("/get-default", handlers.Member.Address.GetDefaultUserAddress)
				addressGroup.GET("/list", handlers.Member.Address.GetAddressList)
			}

			// Point Record (Auth Required)
			pointRecordGroup := memberGroup.Group("/point/record")
			pointRecordGroup.Use(middleware.Auth())
			{
				pointRecordGroup.GET("/page", handlers.Member.PointRecord.GetPointRecordPage)
			}

			// Sign-in Record (App)
			signInGroup := memberGroup.Group("/sign-in/record", middleware.Auth())
			{
				signInGroup.GET("/get-summary", handlers.Member.SignInRecord.GetSignInRecordSummary)
				signInGroup.POST("/create", handlers.Member.SignInRecord.CreateSignInRecord)
				signInGroup.GET("/page", handlers.Member.SignInRecord.GetSignInRecordPage)
			}

			// Sign-in Config (App - Public, 对齐 Java @PermitAll)
			signInConfigGroup := memberGroup.Group("/sign-in/config")
			{
				signInConfigGroup.GET("/list", handlers.Member.SignInConfig.GetSignInConfigList)
			}

			// Social User
			socialUserGroup := memberGroup.Group("/social-user")
			{
				// 公开接口 (对齐 Java @PermitAll)
				socialUserGroup.GET("/get-subscribe-template-list", handlers.Member.SocialUser.GetSubscribeTemplateList)

				// 需要鉴权的接口
				socialUserAuthGroup := socialUserGroup.Group("")
				socialUserAuthGroup.Use(middleware.Auth())
				{
					socialUserAuthGroup.POST("/bind", handlers.Member.SocialUser.Bind)
					socialUserAuthGroup.DELETE("/unbind", handlers.Member.SocialUser.Unbind)
					socialUserAuthGroup.GET("/get", handlers.Member.SocialUser.Get)
					socialUserAuthGroup.POST("/wxa-qrcode", handlers.Member.SocialUser.GetWxaQrcode)
				}
			}
		}

		// ========== Product ==========
		productGroup := appGroup.Group("/product")
		{
			// Category
			categoryGroup := productGroup.Group("/category")
			{
				categoryGroup.GET("/list", handlers.Mall.Product.Category.GetCategoryList)
				categoryGroup.GET("/list-by-ids", handlers.Mall.Product.Category.GetCategoryListByIds)
			}

			// Favorite (Auth Required)
			favoriteGroup := productGroup.Group("/favorite")
			favoriteGroup.Use(middleware.Auth())
			{
				favoriteGroup.POST("/create", handlers.Mall.Product.Favorite.CreateFavorite)
				favoriteGroup.DELETE("/delete", handlers.Mall.Product.Favorite.DeleteFavorite)
				favoriteGroup.GET("/page", handlers.Mall.Product.Favorite.GetFavoritePage)
				favoriteGroup.GET("/exits", handlers.Mall.Product.Favorite.IsFavoriteExists)
				favoriteGroup.GET("/get-count", handlers.Mall.Product.Favorite.GetFavoriteCount)
			}

			// Browse History (Auth Required)
			browseHistoryGroup := productGroup.Group("/browse-history")
			browseHistoryGroup.Use(middleware.Auth())
			{
				browseHistoryGroup.DELETE("/delete", handlers.Mall.Product.BrowseHistory.DeleteBrowseHistory)
				browseHistoryGroup.DELETE("/clean", handlers.Mall.Product.BrowseHistory.CleanBrowseHistory)
				browseHistoryGroup.GET("/page", handlers.Mall.Product.BrowseHistory.GetBrowseHistoryPage)
			}

			// SPU
			spuGroup := productGroup.Group("/spu")
			spuGroup.Use(middleware.ProductErrorHandler()) // 使用商品模块错误处理中间件
			{
				spuGroup.GET("/get-detail", handlers.Mall.Product.Spu.GetSpuDetail)
				spuGroup.GET("/page", handlers.Mall.Product.Spu.GetSpuPage)
				spuGroup.GET("/list-by-ids", handlers.Mall.Product.Spu.GetSpuListByIds)
			}

			// Comment
			commentGroup := productGroup.Group("/comment")
			{
				commentGroup.GET("/page", handlers.Mall.Product.Comment.GetCommentPage)
				commentGroup.POST("/create", middleware.Auth(), handlers.Mall.Product.Comment.CreateComment)
			}
		}

		// ========== Trade ==========
		tradeGroup := appGroup.Group("/trade")
		tradeGroup.Use(middleware.Auth())
		{
			// Cart
			cartGroup := tradeGroup.Group("/cart")
			{
				cartGroup.POST("/add", handlers.Mall.Trade.Cart.AddCart)
				cartGroup.PUT("/update-count", handlers.Mall.Trade.Cart.UpdateCartCount)
				cartGroup.PUT("/update-selected", handlers.Mall.Trade.Cart.UpdateCartSelected)
				cartGroup.PUT("/reset", handlers.Mall.Trade.Cart.ResetCart)
				cartGroup.DELETE("/delete", handlers.Mall.Trade.Cart.DeleteCart)
				cartGroup.GET("/get-count", handlers.Mall.Trade.Cart.GetCartCount)
				cartGroup.GET("/list", handlers.Mall.Trade.Cart.GetCartList)
			}

			// Order
			orderGroup := tradeGroup.Group("/order")
			{
				orderGroup.GET("/settlement", handlers.Mall.Trade.Order.SettlementOrder)
				// settlement-product 移至公开路由组 (对齐 Java @PermitAll)
				orderGroup.POST("/create", handlers.Mall.Trade.Order.CreateOrder)
				orderGroup.GET("/get-detail", handlers.Mall.Trade.Order.GetOrderDetail)
				orderGroup.GET("/item/get", handlers.Mall.Trade.Order.GetOrderItem)
				orderGroup.GET("/page", handlers.Mall.Trade.Order.GetOrderPage)
				orderGroup.GET("/get-count", handlers.Mall.Trade.Order.GetOrderCount)
				orderGroup.PUT("/receive", handlers.Mall.Trade.Order.ReceiveOrder)
				orderGroup.DELETE("/delete", handlers.Mall.Trade.Order.DeleteOrder)
				orderGroup.POST("/item/create-comment", handlers.Mall.Trade.Order.CreateOrderItemComment)
				orderGroup.DELETE("/cancel", handlers.Mall.Trade.Order.CancelOrder)
				orderGroup.GET("/get-express-track-list", handlers.Mall.Trade.Order.GetOrderExpressTrackList)
			}

			// AfterSale
			afterSaleGroup := tradeGroup.Group("/after-sale")
			{
				afterSaleGroup.POST("/create", handlers.Mall.Trade.AfterSale.CreateAfterSale)
				afterSaleGroup.GET("/page", handlers.Mall.Trade.AfterSale.GetAfterSalePage)
				afterSaleGroup.GET("/get", handlers.Mall.Trade.AfterSale.GetAfterSale)
				afterSaleGroup.DELETE("/cancel", handlers.Mall.Trade.AfterSale.CancelAfterSale)
				afterSaleGroup.PUT("/delivery", handlers.Mall.Trade.AfterSale.DeliveryAfterSale)
			}

			// Brokerage User
			brokerageUserGroup := tradeGroup.Group("/brokerage-user")
			{
				brokerageUserGroup.GET("/get", handlers.Mall.Trade.Brokerage.BrokerageUser.GetBrokerageUser)
				brokerageUserGroup.GET("/get-summary", handlers.Mall.Trade.Brokerage.BrokerageUser.GetBrokerageUserSummary)
				brokerageUserGroup.GET("/child-summary-page", handlers.Mall.Trade.Brokerage.BrokerageUser.GetBrokerageUserChildSummaryPage)
				brokerageUserGroup.GET("/get-rank-by-price", handlers.Mall.Trade.Brokerage.BrokerageUser.GetRankByPrice)
				brokerageUserGroup.GET("/rank-page-by-price", handlers.Mall.Trade.Brokerage.BrokerageUser.GetBrokerageUserRankPageByPrice)
				brokerageUserGroup.GET("/rank-page-by-user-count", handlers.Mall.Trade.Brokerage.BrokerageUser.GetBrokerageUserRankPageByUserCount)
				brokerageUserGroup.PUT("/bind", handlers.Mall.Trade.Brokerage.BrokerageUser.BindBrokerageUser)
			}
			brokerageRecordGroup := tradeGroup.Group("/brokerage-record")
			{
				brokerageRecordGroup.GET("/page", handlers.Mall.Trade.Brokerage.BrokerageRecord.GetBrokerageRecordPage)
				brokerageRecordGroup.GET("/get-product-brokerage-price", handlers.Mall.Trade.Brokerage.BrokerageRecord.GetProductBrokeragePrice)
			}
			brokerageWithdrawGroup := tradeGroup.Group("/brokerage-withdraw")
			{
				brokerageWithdrawGroup.GET("/page", handlers.Mall.Trade.Brokerage.BrokerageWithdraw.GetBrokerageWithdrawPage)
				brokerageWithdrawGroup.GET("/get", handlers.Mall.Trade.Brokerage.BrokerageWithdraw.GetBrokerageWithdraw) // GET /get?id=...
				brokerageWithdrawGroup.POST("/create", handlers.Mall.Trade.Brokerage.BrokerageWithdraw.CreateBrokerageWithdraw)
			}
		}

		// ========== Trade (Public) ==========
		// 使用 OptionalAuth 中间件，允许公开访问但尝试解析 Token 以获取登录用户
		tradePublicGroup := appGroup.Group("/trade")
		tradePublicGroup.Use(middleware.OptionalAuth())
		{
			orderPublicGroup := tradePublicGroup.Group("/order")
			{
				// 支付中心 → 商城的业务通知：要求内部调用令牌；
				// 令牌之外，Handler/Processor 仍会重新查询支付单校验状态、金额与商户订单号
				orderPublicGroup.POST("/update-paid",
					middleware.PayNotifyToken(), handlers.Mall.Trade.Order.UpdateOrderPaid)
				orderPublicGroup.GET("/settlement-product", handlers.Mall.Trade.Order.SettlementProduct) // @PermitAll - 获得商品结算信息
			}
		}

		// Trade Config (Public)
		tradeConfigGroup := appGroup.Group("/trade/config")
		{
			tradeConfigGroup.GET("/get", handlers.Mall.Trade.Config.GetTradeConfig)
		}

		// ========== Promotion ==========
		promotionGroup := appGroup.Group("/promotion")
		{
			// Coupon (Auth Required)
			couponGroup := promotionGroup.Group("/coupon")
			couponGroup.Use(middleware.Auth())
			{
				couponGroup.POST("/take", handlers.Mall.Promotion.Coupon.TakeCoupon)
				couponGroup.GET("/page", handlers.Mall.Promotion.Coupon.GetCouponPage)
				couponGroup.GET("/get", handlers.Mall.Promotion.Coupon.GetCoupon)                         // 新增: 获得优惠劵
				couponGroup.GET("/get-unused-count", handlers.Mall.Promotion.Coupon.GetUnusedCouponCount) // 新增: 获得未使用数量
				couponGroup.POST("/match-list", handlers.Mall.Promotion.Coupon.GetCouponMatchList)
			}

			// Coupon Template (Public - 对齐 Java @PermitAll)
			// 使用 OptionalAuth 中间件，允许公开访问但尝试解析 Token 以获取登录用户
			couponTemplateGroup := promotionGroup.Group("/coupon-template")
			couponTemplateGroup.Use(middleware.OptionalAuth())
			{
				couponTemplateGroup.GET("/get", handlers.Mall.Promotion.CouponTemplate.GetCouponTemplate)
				couponTemplateGroup.GET("/list", handlers.Mall.Promotion.CouponTemplate.GetCouponTemplateList)
				couponTemplateGroup.GET("/list-by-ids", handlers.Mall.Promotion.CouponTemplate.GetCouponTemplateListByIds)
				couponTemplateGroup.GET("/page", handlers.Mall.Promotion.CouponTemplate.GetCouponTemplatePage)
			}

			// Banner (Public)
			engine.GET("/app-api/promotion/banner/list", handlers.Mall.Promotion.Banner.GetBannerList)

			// Reward Activity (Public - 对齐 Java @PermitAll)
			engine.GET("/app-api/promotion/reward-activity/get", handlers.Mall.Promotion.RewardActivity.GetRewardActivity)

			// Article (Public)
			articleGroup := promotionGroup.Group("/article")
			{
				articleGroup.GET("/list-category", handlers.Mall.Promotion.Article.GetArticleCategoryList)
				articleGroup.GET("/page", handlers.Mall.Promotion.Article.GetArticlePage)
				articleGroup.GET("/get", handlers.Mall.Promotion.Article.GetArticle)
			}

			// DIY Page (Public)			// DIY
			diyTemplateGroup := promotionGroup.Group("/diy-template")
			{
				diyTemplateGroup.GET("/used", handlers.Mall.Promotion.DiyTemplate.GetUsedDiyTemplate)
				diyTemplateGroup.GET("/get", handlers.Mall.Promotion.DiyTemplate.GetDiyTemplate)
			}
			diyPageGroup := promotionGroup.Group("/diy-page")
			{
				diyPageGroup.GET("/get", handlers.Mall.Promotion.DiyPage.GetDiyPage)
			}

			// Kefu Message
			kefuMessageGroup := promotionGroup.Group("/kefu-message", middleware.Auth())
			{
				kefuMessageGroup.POST("/send", handlers.Mall.Promotion.Kefu.SendMessage)
				kefuMessageGroup.PUT("/update-read-status", handlers.Mall.Promotion.Kefu.UpdateMessageReadStatus)
				kefuMessageGroup.GET("/list", handlers.Mall.Promotion.Kefu.GetMessageList)
			}

			// Activity
			activityGroup := promotionGroup.Group("/activity")
			{
				activityGroup.GET("/list-by-spu-id", handlers.Mall.Promotion.Activity.GetActivityListBySpuId)
			}

			// Combination Activity
			combinationActivityGroup := promotionGroup.Group("/combination-activity")
			{
				combinationActivityGroup.GET("/list-by-ids", handlers.Mall.Promotion.CombinationActivity.GetCombinationActivityListByIds)
				combinationActivityGroup.GET("/get-detail", handlers.Mall.Promotion.CombinationActivity.GetCombinationActivityDetail)
				combinationActivityGroup.GET("/page", handlers.Mall.Promotion.CombinationActivity.GetCombinationActivityPage)
			}

			// Combination Record
			combinationRecordGroup := promotionGroup.Group("/combination-record")
			{
				combinationRecordGroup.GET("/get-summary", handlers.Mall.Promotion.CombinationRecord.GetCombinationRecordSummary)
				combinationRecordGroup.GET("/get-head-list", handlers.Mall.Promotion.CombinationRecord.GetHeadCombinationRecordList)
				combinationRecordGroup.GET("/get-detail", handlers.Mall.Promotion.CombinationRecord.GetCombinationRecordDetail)
				combinationRecordGroup.Use(middleware.Auth())
				combinationRecordGroup.GET("/page", handlers.Mall.Promotion.CombinationRecord.GetCombinationRecordPage)
			}

			// Bargain Activity (Public)
			bargainActivityGroup := promotionGroup.Group("/bargain-activity")
			{
				bargainActivityGroup.GET("/list", handlers.Mall.Promotion.BargainActivity.GetBargainActivityList)
				bargainActivityGroup.GET("/page", handlers.Mall.Promotion.BargainActivity.GetBargainActivityPage)
				bargainActivityGroup.GET("/get-detail", handlers.Mall.Promotion.BargainActivity.GetBargainActivityDetail)
			}

			// Bargain Record
			bargainRecordGroup := promotionGroup.Group("/bargain-record")
			{
				bargainRecordGroup.GET("/get-summary", handlers.Mall.Promotion.BargainRecord.GetBargainRecordSummary)
				bargainRecordGroup.GET("/get-detail", handlers.Mall.Promotion.BargainRecord.GetBargainRecordDetail)
				// Auth Required
				bargainRecordGroup.POST("/create", middleware.Auth(), handlers.Mall.Promotion.BargainRecord.CreateBargainRecord)
			}

			// Bargain Help
			bargainHelpGroup := promotionGroup.Group("/bargain-help")
			{
				bargainHelpGroup.GET("/list", handlers.Mall.Promotion.BargainHelp.GetBargainHelpList)
				bargainHelpGroup.POST("/create", middleware.Auth(), handlers.Mall.Promotion.BargainHelp.CreateBargainHelp)
			}

			// Seckill Activity (Public - 对齐 Java @PermitAll)
			seckillActivityGroup := promotionGroup.Group("/seckill-activity")
			{
				seckillActivityGroup.GET("/get-now", handlers.Mall.Promotion.SeckillActivity.GetNowSeckillActivity)
				seckillActivityGroup.GET("/page", handlers.Mall.Promotion.SeckillActivity.GetSeckillActivityPage)
				seckillActivityGroup.GET("/get", handlers.Mall.Promotion.SeckillActivity.GetSeckillActivity)
				seckillActivityGroup.GET("/get-detail", handlers.Mall.Promotion.SeckillActivity.GetSeckillActivityDetail)
				seckillActivityGroup.GET("/list-by-ids", handlers.Mall.Promotion.SeckillActivity.GetSeckillActivityListByIds)
			}

			seckillConfigGroup := promotionGroup.Group("/seckill-config")
			{
				seckillConfigGroup.GET("/list", handlers.Mall.Promotion.SeckillConfig.GetSeckillConfigList)
			}

			// Point Activity (Public)
			pointActivityGroup := promotionGroup.Group("/point-activity")
			{
				pointActivityGroup.GET("/page", handlers.Mall.Promotion.PointActivity.GetPointActivityPage)
				pointActivityGroup.GET("/get-detail", handlers.Mall.Promotion.PointActivity.GetPointActivity)
				pointActivityGroup.GET("/list-by-ids", handlers.Mall.Promotion.PointActivity.GetPointActivityListByIds)
			}
		}

		// ========== Pay ==========
		payGroup := appGroup.Group("/pay")
		payGroup.Use(middleware.Auth())
		{
			// Order
			orderGroup := payGroup.Group("/order")
			{
				orderGroup.GET("/get", handlers.Pay.Order.GetOrder)
				orderGroup.POST("/submit", handlers.Pay.Order.Submit)
			}
			// Wallet
			walletGroup := payGroup.Group("/wallet")
			{
				walletGroup.GET("/get", handlers.Pay.Wallet.GetWallet)
			}
			// Wallet Transaction
			walletTransactionGroup := payGroup.Group("/wallet-transaction")
			{
				walletTransactionGroup.GET("/page", handlers.Pay.WalletTransaction.GetWalletTransactionPage)
				walletTransactionGroup.GET("/get-summary", handlers.Pay.WalletTransaction.GetWalletTransactionSummary)
			}
			// Wallet Recharge
			rechargeGroup := payGroup.Group("/wallet-recharge")
			{
				rechargeGroup.POST("/create", handlers.Pay.Wallet.CreateRecharge)
				rechargeGroup.GET("/page", handlers.Pay.Wallet.GetRechargePage)
			}
			// Wallet Recharge Package
			rechargePackageGroup := payGroup.Group("/wallet-recharge-package")
			{
				rechargePackageGroup.GET("/list", handlers.Pay.WalletRechargePackage.GetWalletRechargePackageList)
			}
			// Channel
			channelGroup := payGroup.Group("/channel")
			{
				channelGroup.GET("/get-enable-code-list", handlers.Pay.Channel.GetEnableChannelCodeList)
			}
			// Transfer
			transferGroup := payGroup.Group("/transfer")
			{
				transferGroup.GET("/sync", handlers.Pay.Transfer.SyncTransfer)
			}
		}
	}
}
