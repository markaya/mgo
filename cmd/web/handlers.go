package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/validator"
	"github.com/markaya/meinappf/ui"
	"golang.org/x/crypto/bcrypt"
)

type accountCreateForm struct {
	AccountName string
	Currency    int
	validator.Validator
}

type userSignupForm struct {
	Name     string
	Email    string
	Password string
	validator.Validator
}

type userLoginForm struct {
	Email    string
	Password string
	validator.Validator
}

type changePasswordForm struct {
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
	validator.Validator
}

type transferCreateForm struct {
	FromAcc    models.Account
	FromAmount float64
	ToAcc      models.Account
	ToAmount   float64
	Date       string
	validator.Validator
}

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
	data.DateString = time.Now().Format("2006-01-02")
	data.Form = transferCreateForm{}
	app.render(w, http.StatusOK, "transfer_create.tmpl.html", data)
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
		app.clientError(w, http.StatusBadRequest)
		return
	}

	fromAccId, err := strconv.Atoi(r.PostForm.Get("from"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	toAccId, err := strconv.Atoi(r.PostForm.Get("to"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	fromAmount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	_, err = time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	fromAcc, err := app.accounts.Get(userId, fromAccId)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	toAcc, err := app.accounts.Get(userId, toAccId)
	if err != nil {
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

	form := transferCreateForm{
		FromAcc:    *fromAcc,
		FromAmount: fromAmount,
		ToAcc:      *toAcc,
		ToAmount:   toAmount,
		Date:       r.PostForm.Get("date"),
	}

	data := app.newTemplateData(r)
	data.Form = form
	form.CheckField(validator.GreaterThanZero(form.FromAmount), "amount", "This field must be greater than zero.")

	if fromAcc.Balance < fromAmount {
		form.AddFieldError("amount", "Account does not have suficient funds.")
	}

	if !form.Valid() {
		data := app.newTemplateData(r)
		app.render(w, http.StatusUnprocessableEntity, "transfer_create.tmpl.html", data)
		return
	}

	confirmed, err := strconv.ParseBool(r.PostForm.Get("confirm"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	if confirmed {
		// TODO: save transfer
	} else {
		ts, err := template.ParseFS(ui.Files, "html/pages/transfer_confirm.tmpl.html")
		if err != nil {
			app.serverError(w, err)
			return
		}
		ts.Execute(w, data)
	}
}

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
	data.DateString = time.Now().Format("2006-01-02")

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
		app.clientError(w, http.StatusBadRequest)
		return
	}

	accId, err := strconv.Atoi(r.PostForm.Get("account"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	account, err := app.accounts.Get(userId, accId)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currency := account.Currency

	txType, err := strconv.Atoi(r.PostForm.Get("txtype"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	_, err = time.Parse("2006-01-02", r.PostForm.Get("date"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := models.TransactionCreateForm{
		UserId:          userId,
		AccountId:       account.ID,
		Date:            r.PostForm.Get("date"),
		Amount:          amount,
		Category:        r.PostForm.Get("category"),
		Description:     r.PostForm.Get("description"),
		Currency:        int(currency),
		TransactionType: txType,
	}

	// TODO: Check if more is needed.
	form.CheckField(validator.NotBlank(form.Date), "date", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Category), "category", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Category, 25), "category", "This field cannto be more than 25 chars long.")
	form.CheckField(validator.MaxChars(form.Description, 100), "descritpion", "This field cannto be more than 100 chars long.")
	form.CheckField(validator.PermittedInt(form.Currency, 0, 1), "currency", "This field must equal 0(RSD) or 1(EUR)")
	form.CheckField(validator.PermittedInt(form.TransactionType, 0, 1), "txtype", "This field must equal 0(INCOME) or 1(EXPENSE)")
	form.CheckField(validator.GreaterThanZero(form.Amount), "amount", "This field must be greater than zero.")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "transactionCreate.tmpl.html", data)
		return
	}

	txAmountSigned := amount
	var transactionType = models.TransactionType(txType)

	if transactionType == models.Expense {
		txAmountSigned = -formAmount
	}

	newBalance := account.Balance + txAmountSigned
	if newBalance < 0 {
		form.AddFieldError("amount", "Account does not have suficient funds.")
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
			app.serverError(w, err)
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Account successfully created!")
	http.Redirect(w, r, "/transaction/create", http.StatusSeeOther)
}

func (app *application) transactionsView(w http.ResponseWriter, r *http.Request) {
	app.notFound(w)
}

func (app *application) transactionView(w http.ResponseWriter, r *http.Request) {
	app.notFound(w)
}

func (app *application) accountCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)

	data.Form = accountCreateForm{Currency: 0}
	app.render(w, http.StatusOK, "accountCreate.tmpl.html", data)
}

func (app *application) accountCreatePost(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	currency, err := strconv.Atoi(r.PostForm.Get("currency"))
	if err != nil {
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
		app.render(w, http.StatusUnprocessableEntity, "accountCreate.tmpl.html", data)
		return
	}

	id, err := app.accounts.Insert(userId, form.AccountName, models.Currency(form.Currency))
	if err != nil {
		if errors.Is(err, models.ErrDuplicateAccountName) {
			form.AddFieldError("name", "Account name already in use.")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "accountCreate.tmpl.html", data)
		} else {
			app.serverError(w, err)
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Account successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/account/view/%d", id), http.StatusSeeOther)
}

func (app *application) accountsView(w http.ResponseWriter, r *http.Request) {
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

	data := app.newTemplateData(r)
	data.Accounts = accounts
	app.render(w, http.StatusOK, "accounts.tmpl.html", data)
}

func (app *application) accountView(w http.ResponseWriter, r *http.Request) {
	userId := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if userId == 0 {
		err := errors.New("Unauthorized user requesting account view.")
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

	data := app.newTemplateData(r)
	data.Account = account
	app.render(w, http.StatusOK, "account.tmpl.html", data)
}

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

	incomeTransactions, err := app.transactions.GetLatest(userId, 10, models.Income)
	if err != nil {
		app.serverError(w, err)
		return
	}
	expenseTransactions, err := app.transactions.GetLatest(userId, 10, models.Expense)
	if err != nil {
		app.serverError(w, err)
		return
	}

	data.IncomeTransactions = incomeTransactions
	data.ExpenseTransactions = expenseTransactions

	app.render(w, http.StatusOK, "home.tmpl.html", data)
}

func (app *application) aboutView(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "about.tmpl.html", data)
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.tmpl.html", data)
}
func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := userSignupForm{
		Name:     r.PostForm.Get("name"),
		Email:    r.PostForm.Get("email"),
		Password: r.PostForm.Get("password"),
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be valid email adress")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 chars long")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.tmpl.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Your signup was successfull. Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.tmpl.html", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := userLoginForm{
		Email:    r.PostForm.Get("email"),
		Password: r.PostForm.Get("password"),
	}

	form.CheckField(validator.NotBlank(form.Email), "name", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be valid email adress")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.tmpl.html", data)
		} else {
			app.serverError(w, err)
		}
		return
	}
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserId", id)

	redirectPath := app.sessionManager.PopString(r.Context(), "redirectPathAfterLogIn")
	if redirectPath != "" {
		http.Redirect(w, r, redirectPath, http.StatusSeeOther)
	} else {

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

}
func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Remove(r.Context(), "authenticatedUserId")

	app.sessionManager.Put(r.Context(), "flash", "You have been logged out successfully")

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
	//http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userView(w http.ResponseWriter, r *http.Request) {
	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if id == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}
	user, err := app.users.Get(id)
	if err != nil {
		err := fmt.Errorf("Authenticated user with %d does not exist in DB", id)
		app.serverError(w, err)
		return
	}

	accounts, err := app.accounts.GetAll(id)
	if err != nil {
		app.errorLog.Println("error while getting all accounts for user")
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.User = user
	data.Accounts = accounts
	app.render(w, http.StatusOK, "userAccount.tmpl.html", data)

}

func (app *application) accountPasswordUpdate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = changePasswordForm{}

	app.render(w, http.StatusOK, "password.tmpl.html", data)
}
func (app *application) accountPasswordUpdatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := changePasswordForm{
		CurrentPassword:    r.PostForm.Get("currentPassword"),
		NewPassword:        r.PostForm.Get("newPassword"),
		NewPasswordConfirm: r.PostForm.Get("newPasswordConfirmation"),
	}

	form.CheckField(validator.NotBlank(form.CurrentPassword), "currentPassword", "Current password field must not empty.")
	form.CheckField(validator.NotBlank(form.NewPassword), "currentPassword", "New password field must not empty.")
	form.CheckField(validator.NotBlank(form.NewPasswordConfirm), "newPasswordConfirmation", "Confirm password field must not empty.")
	form.CheckField(validator.MinChars(form.NewPassword, 8), "newPassword", "This field must be at least 8 chars long")
	form.CheckField(validator.MinChars(form.NewPasswordConfirm, 8), "newPasswordConfirmation", "This field must be at least 8 chars long")

	id := app.sessionManager.GetInt(r.Context(), "authenticatedUserId")
	if id == 0 {
		err := errors.New("Unauthorized user requesting account view.")
		app.serverError(w, err)
		return
	}
	user, err := app.users.Get(id)
	if err != nil {
		err := fmt.Errorf("Authenticated user with %d does not exist in DB", id)
		app.serverError(w, err)
		return
	}

	match := true
	err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(form.CurrentPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			match = false
		} else {
			app.serverError(w, err)
			return
		}
	}

	form.CheckField(match, "currentPassword", "Wrong password!")
	form.CheckField(form.NewPassword == form.NewPasswordConfirm, "newPassword", "New passwords do not match!")
	form.CheckField(form.NewPassword == form.NewPasswordConfirm, "newPasswordConfirmation", "New passwords do not match!")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "password.tmpl.html", data)
		return
	}

	// FIXME: When inverted id and password werte sent to db stmt then no error
	// occured and i had no idea where bug was
	err = app.users.UpdatePassword(id, form.NewPassword)
	if err != nil {
		app.serverError(w, err)
		return
	}

	//TODO: Add flash for "successfully changed password"
	app.sessionManager.Put(r.Context(), "flash", "Successfully changed password!")

	http.Redirect(w, r, "/user/view/", http.StatusSeeOther)
}
