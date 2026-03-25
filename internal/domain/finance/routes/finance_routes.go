package routes

import (
	"life-tracker-backend/internal/domain/finance/controller"
	"life-tracker-backend/internal/domain/finance/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func RegisterFinanceRoutes(router *gin.RouterGroup, mongoDB *mongo.Database) {
	financeService := service.NewFinanceService(mongoDB)
	financeController := controller.NewFinanceController(financeService)

	financeGroup := router.Group("/finances")
	{
		financeGroup.GET("/categories", financeController.GetCategories)

		financeGroup.POST("/transactions", financeController.CreateTransaction)
		financeGroup.GET("/transactions", financeController.GetTransactions)
		financeGroup.GET("/transactions/:id", financeController.GetTransaction)
		financeGroup.PUT("/transactions/:id", financeController.UpdateTransaction)
		financeGroup.DELETE("/transactions/:id", financeController.DeleteTransaction)

		financeGroup.GET("/fixed-transactions", financeController.GetFixedTransactions)
		financeGroup.GET("/fixed-transactions/:id", financeController.GetFixedTransactionWithPayments)

		financeGroup.POST("/payments", financeController.CreatePayment)
		financeGroup.GET("/payments", financeController.GetPayments)
		financeGroup.DELETE("/payments/:id", financeController.DeletePayment)

		financeGroup.GET("/summary", financeController.GetFinanceSummary)
		financeGroup.GET("/monthly-stats", financeController.GetMonthlyStats)
	}
}
