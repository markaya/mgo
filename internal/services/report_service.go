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

func GetTotalReport(transactions []*models.Transaction, startDate, endDate time.Time) TotalReport {
	incomeTransactions := make([]models.Transaction, 100)
	expenseTransactions := make([]models.Transaction, 100)
	incomeEur := 0.0
	incomeRsd := 0.0
	expenseEur := 0.0
	expenseRsd := 0.0

	for _, v := range transactions {
		// NOTE: Ignore transfer
		switch v.TransactionType {
		case models.Income:
			switch v.Currency {
			case models.Euro:
				incomeEur += v.Amount
			case models.SerbianDinar:
				incomeRsd += v.Amount
			default:
				panic("unsupported currency")
			}
			incomeTransactions = append(incomeTransactions, *v)
		case models.Expense:
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
		StartDate:           startDate,
		EndDate:             endDate,
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
