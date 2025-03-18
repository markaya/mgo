package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

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

	app.sessionManager.Put(r.Context(), "flash", "Successfully changed password!")
	http.Redirect(w, r, "/user/view/", http.StatusSeeOther)
}
