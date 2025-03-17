package services

import (
	"math"
	"time"

	"github.com/markaya/meinappf/internal/models"
)

type TotalReport struct {
	StartDate           time.Time
	EndDate             time.Time
	IncomeEur           float64
	ExpenseEur          float64
	ProgressEur         int
	IncomeRsd           float64
	ExpenseRsd          float64
	ProgressRsd         int
	IncomeTransactions  []models.Transaction
	ExpenseTransactions []models.Transaction
}

func GetTotalReport(transactions []*models.Transaction) TotalReport {
	/*TODO:
	  1. Separate income and expenes, avoid transfers?? Leave transfers for transfers
	  2. Sum Income
	  3. Sum Expense
	  4. Calculte net Profit/Loss
	*/
	incomeTransactions := make([]models.Transaction, 100)
	expenseTransactions := make([]models.Transaction, 100)
	incomeEur := 0.0
	incomeRsd := 0.0
	expenseEur := 0.0
	expenseRsd := 0.0

	for _, v := range transactions {
		// NOTE: Ignore transfer
		if v.TransactionType == models.Income {
			switch v.Currency {
			case models.Euro:
				incomeEur += v.Amount
			case models.SerbianDinar:
				incomeRsd += v.Amount
			default:
				panic("unsupported currency")
			}
			incomeTransactions = append(incomeTransactions, *v)
		} else if v.TransactionType == models.Expense {
			switch v.Currency {
			case models.Euro:
				expenseEur += v.Amount
			case models.SerbianDinar:
				expenseRsd += v.Amount
			default:
				panic("unsupported currency")
			}
			expenseTransactions = append(expenseTransactions, *v)
		}
	}

	progressEur := 0
	if incomeEur > 0 {
		progressEur = int(math.Round((expenseEur / incomeEur) * 100))
	}
	progressRsd := 0
	if incomeRsd > 0 {
		progressRsd = int(math.Round((expenseRsd / incomeRsd) * 100))
	}

	return TotalReport{
		StartDate:           time.Now(),
		EndDate:             time.Now(),
		IncomeEur:           incomeEur,
		ExpenseEur:          expenseEur,
		ProgressEur:         progressEur,
		IncomeRsd:           incomeRsd,
		ExpenseRsd:          expenseRsd,
		ProgressRsd:         progressRsd,
		IncomeTransactions:  incomeTransactions,
		ExpenseTransactions: expenseTransactions,
	}
}
