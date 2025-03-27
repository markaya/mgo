package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/validator"
)

type accountCreateForm struct {
	AccountName string
	Currency    int
	validator.Validator
}

type rebalanceAccountForm struct {
	accountId  int
	newBalance float64
	validator.Validator
}

func (app *application) accountCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	data.Form = accountCreateForm{Currency: 0}
	app.render(w, http.StatusOK, "account_create.html", data)
}

func (app *application) accountCreatePost(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("unauthorized user requesting account view")
		app.serverError(w, err)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Printf("could not parse form")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currency, err := strconv.Atoi(r.PostForm.Get("currency"))
	if err != nil {
		app.errorLog.Printf("could not parse form currency")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := accountCreateForm{
		AccountName: r.PostForm.Get("name"),
		Currency:    currency,
	}

	form.CheckField(validator.NotBlank(form.AccountName), "name", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.AccountName, 20), "name", "This field cannto be more than 20 chars long.")
	form.CheckField(validator.PermittedInt(form.Currency, 0, 1), "currency", "This field must equal 0(RSD) or 1(EUR)")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "account_create.html", data)
		return
	}

	id, err := app.accounts.Insert(userId, form.AccountName, models.Currency(form.Currency))
	if err != nil {
		if errors.Is(err, models.ErrDuplicateAccountName) {
			form.AddFieldError("name", "Account name already in use.")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "account_create.html", data)
		} else {
			app.serverError(w, err)
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Account successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/account/view/%d", id), http.StatusSeeOther)
}

func (app *application) accountsView(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("unauthorized user requesting account view")
		app.serverError(w, err)
		return
	}
	user, err := app.users.Get(userId)
	if err != nil {
		app.errorLog.Printf("could not find user with id %d", userId)
		app.serverError(w, err)
		return
	}

	accounts, err := app.accounts.GetAll(userId)
	if err != nil {
		app.errorLog.Println("error while getting all accounts for user")
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Accounts = accounts
	data.User = user
	app.render(w, http.StatusOK, "accounts.html", data)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("unauthorized user requesting account view")
		app.serverError(w, err)
		return
	}

	rawID := r.PathValue("id")

	accoutnId, err := strconv.Atoi(rawID)
	if err != nil || accoutnId < 1 {
		app.notFound(w)
		return
	}

	account, err := app.accounts.Get(userId, accoutnId)
	if err != nil || account == nil {
		app.errorLog.Printf("account for user %d, with account id %d, does not exist", userId, accoutnId)
		app.notFound(w)
		return
	}

	user, ok := r.Context().Value(authenticatedUser).(*models.User)
	if !ok {
		err := errors.New("unauthorized user requesting account view, could not find user in context")
		app.serverError(w, err)
		return

	}

	data := app.newTemplateData(r)
	data.User = user
	data.Account = account
	app.render(w, http.StatusOK, "account.html", data)
}

func (app *application) accountRebalanceView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	accIdRaw := r.PathValue("id")
	accountId, err := strconv.Atoi(accIdRaw)
	if err != nil {
		app.errorLog.Printf("error parsing id %s into Integer type.", accIdRaw)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")

	account, err := app.accounts.Get(userId, accountId)
	if err != nil {
		app.errorLog.Printf("could not find account for user %d, with id %d", userId, accountId)
		return
	}
	data.Account = account
	form := rebalanceAccountForm{}
	data.Form = form

	app.render(w, http.StatusOK, "rebalance.html", data)
}
func (app *application) accountRebalancePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Printf("Error parsing form")
		app.clientError(w, http.StatusBadRequest)
		return
	}
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")

	accId, err := strconv.Atoi(r.PostForm.Get("id"))
	if err != nil {
		app.errorLog.Printf("error parsing acc id %s", r.PostForm.Get("id"))
		app.clientError(w, http.StatusBadRequest)
		return
	}

	newBalance, err := strconv.ParseFloat(r.PostForm.Get("new-balance"), 64)
	if err != nil {
		app.errorLog.Printf("error parsing new-balance %s", r.PostForm.Get("new-balance"))
		app.clientError(w, http.StatusBadRequest)
		return
	}

	acc, err := app.accounts.Get(userId, accId)
	if err != nil {
		app.errorLog.Printf("could not find account with id %d, for user %d", accId, userId)
		app.clientError(w, http.StatusBadRequest)
		return
	}
	data := app.newTemplateData(r)
	form := rebalanceAccountForm{
		accountId:  accId,
		newBalance: newBalance,
	}

	form.CheckField(validator.GreaterThanZero(form.newBalance), "balance", "This field must be greater than zero.")

	balanceDiff := acc.Balance - newBalance
	if balanceDiff == 0 {
		form.AddFieldError("balance", "New balance can not be same as current balance")
	}

	data.Form = form
	if !form.Valid() {
		app.render(w, http.StatusUnprocessableEntity, "rebalance.html", data)
		return
	}

	transactionCreateForm := models.NewRebalance(*acc, balanceDiff)
	_, err = app.transactions.Insert(transactionCreateForm, newBalance)
	if err != nil {
		app.errorLog.Printf("could not insert transaction create form %v", transactionCreateForm)
		app.serverError(w, err)
		return
	}

	acc.Balance = newBalance
	data.Account = acc
	app.sessionManager.Put(r.Context(), "flash", "Account successfully rebalanced!")
	http.Redirect(w, r, fmt.Sprintf("/account/view/%d", acc.ID), http.StatusSeeOther)
}
