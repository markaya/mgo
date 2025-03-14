package services

import (
	"time"

	"github.com/markaya/meinappf/internal/models"
)

type TotalReport struct {
	userId              int
	startDate           time.Time
	endDate             time.Time
	income              float64
	expense             float64
	incomeTransactions  []models.Transaction
	expenseTransactions []models.Transaction
}

func GetTotalReport(transactions []models.Transaction) TotalReport {
	/*TODO:
	  1. Separate income and expenes, avoid transfers?? Leave transfers for transfers
	  2. Sum Income
	  3. Sum Expense
	  4. Calculte net Profit/Loss
	*/

	return TotalReport{}
}
