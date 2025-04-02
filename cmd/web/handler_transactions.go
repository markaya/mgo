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
		err := fmt.Errorf("path %s does not exist", path)
		app.serverError(w, err)
		return
	}

	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if id == 0 {
		err := errors.New("unauthorized user requesting account view")
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

	app.render(w, http.StatusOK, "transaction_create.html", data)
}

func (app *application) transactionCreatePost(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("unauthorized user requesting account view")
		app.serverError(w, err)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.infoLog.Println("error while parsing form")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	accId, err := strconv.Atoi(r.PostForm.Get("account"))
	if err != nil {
		app.infoLog.Println("error while parsing account id")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	account, err := app.accounts.Get(userId, accId)
	if err != nil {
		app.infoLog.Println("error while getting account from database")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currency := account.Currency

	txType, err := strconv.Atoi(r.PostForm.Get("txtype"))
	if err != nil {
		app.infoLog.Println("error while parsing transaction type")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 64)
	if err != nil {
		app.infoLog.Println("error while parsing amount type")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.infoLog.Println("error while parsing date")
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

		switch transactionType {
		case models.Expense:
			data.DefaultExpenseCategories()
		case models.Income:
			data.DefaultIncomeCategories()
		default:
			app.errorLog.Println("unsuported transaction type for user creation of transaction")
			panic("unsupported tx type")

		}

		data.Accounts = accounts
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "transaction_create.html", data)
		return
	}

	// FIXME: sending account like this is prime call for race conditions.
	// It is fine for now as there is no concurrent writes.
	_, err = app.transactions.Insert(form, newBalance)
	if err != nil {
		if errors.Is(err, models.ErrAccountDoesNotExist) {
			form.AddFieldError("account", "Account does not exist.")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "account_create.html", data)
		} else {
			app.infoLog.Println("server error when inserting!")
			app.serverError(w, err)
			return
		}
	}

	var redirectUrl string
	switch transactionType {
	case models.Expense:
		redirectUrl = "/transaction/create/expense"
	case models.Income:
		redirectUrl = "/transaction/create/income"
	default:
		redirectUrl = "/"

	}

	app.sessionManager.Put(r.Context(), "flash", "Transaction successfully created!")
	http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
}

func (app *application) transactionsView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.WithDefaultDateFilter()

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		app.infoLog.Printf("could not find user with id %d", userId)
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	user, err := app.users.Get(userId)
	if err != nil {
		app.errorLog.Printf("could not find user with id %d", userId)
		app.serverError(w, err)
		return
	}
	data.User = user

	err = r.ParseForm()
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

	incomeTransactions, err := app.transactions.GetByDateAndType(
		userId,
		models.Income,
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
	if err != nil {
		app.serverError(w, err)
		return
	}

	report := services.GetTotalReport(
		append(incomeTransactions, expenseTransactions...),
		data.DateFilter["startDate"],
		data.DateFilter["endDate"],
	)

	data.UserTotalReport = report
	data.IncomeTransactions = incomeTransactions
	data.ExpenseTransactions = expenseTransactions

	app.render(w, http.StatusOK, "transactions.html", data)
}

func (app *application) groupingsView(w http.ResponseWriter, r *http.Request) {

	data := app.newTemplateData(r)
	data.WithDefaultDateFilter()

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		app.infoLog.Printf("could not find user with id %d", userId)
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	user, err := app.users.Get(userId)
	if err != nil {
		app.errorLog.Printf("could not find user with id %d", userId)
		app.serverError(w, err)
		return
	}
	data.User = user

	err = r.ParseForm()
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

	groupings, err := app.transactions.GetGroupingByDate(
		userId,
		data.DateFilter["startDate"],
		data.DateFilter["endDate"],
	)

	if err != nil {
		app.errorLog.Printf("could not fetch grouping for user %d\n", userId)
		app.serverError(w, err)
		return
	}
	data.GroupingReports = groupings

	app.render(w, http.StatusOK, "groupings.html", data)
}
