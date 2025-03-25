package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/validator"
)

// TODO: Maybe later to show all transfers as they are not shown with others
func (app *application) transferView(w http.ResponseWriter, r *http.Request) {
	app.notFound(w)
}

func (app *application) transfersView(w http.ResponseWriter, r *http.Request) {
	app.notFound(w)
}

func (app *application) transferCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
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
	data.DateStringNow = time.Now().Format("2006-01-02")
	data.Form = models.TransferCreateForm{}
	app.render(w, http.StatusOK, "transfer_create.html", data)
}

func (app *application) transferCreatePost(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Printf("error parsing form")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	fromAccId, err := strconv.Atoi(r.PostForm.Get("from"))
	if err != nil {
		app.errorLog.Printf("error parsing from acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	toAccId, err := strconv.Atoi(r.PostForm.Get("to"))
	if err != nil {
		app.errorLog.Printf("error parsing to acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	fromAmount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 64)
	if err != nil {
		app.errorLog.Printf("error parsing from amount acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.errorLog.Printf("error parsing date acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	fromAcc, err := app.accounts.Get(userId, fromAccId)
	if err != nil {
		app.errorLog.Printf("error parsing from  acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}
	toAcc, err := app.accounts.Get(userId, toAccId)
	if err != nil {
		app.errorLog.Printf("error parsing to  acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	toAmount := 0.0
	if fromAcc.Currency == toAcc.Currency {
		toAmount = fromAmount
	} else {
		if fromAcc.Currency == models.Euro {
			toAmount = fromAmount * 117
		} else {
			toAmount = fromAmount / 117
		}
	}

	form := models.TransferCreateForm{
		FromAcc:    *fromAcc,
		FromAmount: fromAmount,
		ToAcc:      *toAcc,
		ToAmount:   toAmount,
		Date:       date,
	}

	data := app.newTemplateData(r)
	form.CheckField(validator.GreaterThanZero(form.FromAmount), "amount", "This field must be greater than zero.")

	if fromAcc.Balance < fromAmount {
		form.AddFieldError("amount", "Account does not have suficient funds.")
	}

	if fromAccId == toAccId {
		form.AddFieldError("from", "Trying to transfer funds from one account to itself.")
		form.AddFieldError("to", "Trying to transfer funds from one account to itself.")
	}

	data.Form = form
	if !form.Valid() {
		accounts, err := app.accounts.GetAll(userId)
		if err != nil {
			app.errorLog.Println("error while getting all accounts for user")
			app.serverError(w, err)
			return
		}

		data.Accounts = accounts
		app.renderForm(w, http.StatusOK, "transfer_create_form.html", "transfer-create-form", data)
		return
	}

	confirmed, err := strconv.ParseBool(r.PostForm.Get("confirm"))
	if err != nil {
		app.errorLog.Printf("Error parsing to  acc")
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if confirmed {
		err = app.transactions.InsertTransfer(form)
		if err != nil {
			app.serverError(w, err)
			return
		}

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)

	} else {
		app.renderForm(w, http.StatusOK, "transfer_confirm.tmpl.html", "transfer-confirm", data)
	}
}
