package main

import (
	"net/http"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/services"
)

func (app *application) ping(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("OK"))
	if err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		// TODO: serve unauthenticated home page
		app.render(w, http.StatusOK, "home.tmpl.html", data)
	}
	data.WithDefaultDateFilter()

	allTransactions, err := app.transactions.GetByDate(userId, data.DateFilter["startDate"], data.DateFilter["endDate"])
	if err != nil {
		app.errorLog.Println("error while getting transactions")
		app.serverError(w, err)
		return
	}
	report := services.GetTotalReport(allTransactions)

	incomeTransactions, err := app.transactions.GetLatest(userId, 5, models.Income)
	if err != nil {
		app.errorLog.Println("error while getting income transactions")
		app.serverError(w, err)
		return
	}
	expenseTransactions, err := app.transactions.GetLatest(userId, 5, models.Expense)
	if err != nil {
		app.errorLog.Println("error while getting expense transactions")
		app.serverError(w, err)
		return
	}

	data.IncomeTransactions = incomeTransactions
	data.ExpenseTransactions = expenseTransactions
	data.UserTotalReport = report

	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) aboutView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "about.tmpl.html", data)
}
