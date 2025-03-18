package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/services"
	"github.com/markaya/meinappf/internal/validator"
)

func (app *application) transactionCreate(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("ttype")

	data := app.newTemplateData(r)
	var transactionType models.TransactionType
	switch path {
	case "income":
		transactionType = models.Income
		data.DefaultIncomeCategories()
	case "expense":
		transactionType = models.Expense
		data.DefaultExpenseCategories()
	default:
		err := fmt.Errorf("Path %s does not exist.", path)
		app.serverError(w, err)
		return
	}

	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if id == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}

	accounts, err := app.accounts.GetAll(id)
	if err != nil {
		app.errorLog.Println("error while getting all accounts for user")
		app.serverError(w, err)
		return
	}

	data.Accounts = accounts
	data.Form = models.TransactionCreateForm{Currency: 0, TransactionType: int(transactionType)}
	data.DateStringNow = time.Now().Format("2006-01-02")

	app.render(w, http.StatusOK, "transactionCreate.tmpl.html", data)
}

func (app *application) transactionCreatePost(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}

	err := r.ParseForm()
	// FIXME: Fix these client errors to more user friendly error handling.
	// Use only for those who should not be altered with
	if err != nil {
		app.infoLog.Println("error while parsing form!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	accId, err := strconv.Atoi(r.PostForm.Get("account"))
	if err != nil {
		app.infoLog.Println("error while parsing account id!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	account, err := app.accounts.Get(userId, accId)
	if err != nil {
		app.infoLog.Println("error while getting account from database!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currency := account.Currency

	txType, err := strconv.Atoi(r.PostForm.Get("txtype"))
	if err != nil {
		app.infoLog.Println("error while parsing transaction type!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 64)
	if err != nil {
		app.infoLog.Println("error while parsing amount type!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.infoLog.Println("error while parsing date!")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := models.TransactionCreateForm{
		UserId:          userId,
		AccountId:       account.ID,
		Date:            date,
		Amount:          amount,
		Category:        r.PostForm.Get("category"),
		Description:     r.PostForm.Get("description"),
		Currency:        int(currency),
		TransactionType: txType,
	}

	// TODO: Check if more is needed.
	form.CheckField(validator.NotBlank(form.Category), "category", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Category, 25), "category", "This field cannto be more than 25 chars long.")
	form.CheckField(validator.MaxChars(form.Description, 100), "descritpion", "This field cannto be more than 100 chars long.")
	form.CheckField(validator.PermittedInt(form.Currency, 0, 1), "currency", "This field must equal 0(RSD) or 1(EUR)")
	form.CheckField(validator.PermittedInt(form.TransactionType, 0, 1), "txtype", "This field must equal 0(INCOME) or 1(EXPENSE)")
	form.CheckField(validator.GreaterThanZero(form.Amount), "amount", "This field must be greater than zero.")

	txAmountSigned := amount
	var transactionType = models.TransactionType(txType)

	if transactionType == models.Expense {
		txAmountSigned = -amount
	}

	newBalance := account.Balance + txAmountSigned
	if newBalance < 0 {
		form.AddFieldError("amount", "Account does not have suficient funds.")
	}

	if !form.Valid() {
		accounts, err := app.accounts.GetAll(userId)
		if err != nil {
			app.errorLog.Println("error while getting all accounts for user")
			app.serverError(w, err)
			return
		}
		data := app.newTemplateData(r)

		if transactionType == models.Expense {
			data.DefaultExpenseCategories()
		} else if transactionType == models.Income {
			data.DefaultIncomeCategories()
		} else {
			panic("unsupported tx type")
		}

		data.Accounts = accounts
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "transactionCreate.tmpl.html", data)
		return
	}

	// FIXME: sending account like this is prime call for race conditions.
	// It is fine for now as there is no concurrent writes.
	_, err = app.transactions.Insert(form, newBalance)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateAccountName) {
			form.AddFieldError("account", "Account does not exist.")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "accountCreate.tmpl.html", data)
		} else {
			app.infoLog.Println("server error when inserting!")
			app.serverError(w, err)
			return
		}
	}

	var redirectUrl string
	if transactionType == models.Expense {
		redirectUrl = "/transaction/create/expense"
	} else if transactionType == models.Income {
		redirectUrl = "/transaction/create/income"
	} else {
		redirectUrl = "/"
	}
	app.sessionManager.Put(r.Context(), "flash", "Transaction successfully created!")
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

func (app *application) transactionsView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.WithDefaultDateFilter()

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println("err while parsing form")
		app.serverError(w, err)
		return
	}

	startDateString := r.Form.Get("start-date")
	endDateString := r.Form.Get("end-date")

	if startDateString != "" {
		startDate, err := time.Parse("2006-01-02", startDateString)
		if err == nil {
			data.DateFilter["startDate"] = startDate
		}
	}
	if endDateString != "" {
		endDate, err := time.Parse("2006-01-02", endDateString)
		if err == nil {
			data.DateFilter["endDate"] = endDate
		}
	}

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		// TODO: serve unauthenticated home page
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	incomeTransactions, err := app.transactions.GetByDateAndType(
		userId,
		models.Income,
		// FIXME: Should you gamble with this? I mean I know that there is default filter,
		// but there is maybe future issue?
		data.DateFilter["startDate"],
		data.DateFilter["endDate"],
	)
	if err != nil {
		app.serverError(w, err)
		return
	}

	expenseTransactions, err := app.transactions.GetByDateAndType(
		userId,
		models.Expense,
		data.DateFilter["startDate"],
		data.DateFilter["endDate"],
	)

	report := services.GetTotalReport(append(incomeTransactions, expenseTransactions...))

	if err != nil {
		app.serverError(w, err)
		return
	}

	data.UserTotalReport = report
	data.IncomeTransactions = incomeTransactions
	data.ExpenseTransactions = expenseTransactions

	app.render(w, http.StatusOK, "transactions.tmpl.html", data)
}
