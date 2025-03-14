package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/ui"
)

// TODO: Add success and fail flash.
type templateData struct {
	CurrentYear         int
	DateString          string
	Form                any
	Flash               string
	IsAuthenticated     bool
	User                *models.User
	Account             *models.Account
	Accounts            []*models.Account
	Categories          []string
	IncomeTransactions  []*models.Transaction
	ExpenseTransactions []*models.Transaction
}

func (t *templateData) DefaultIncomeCategories() {
	t.Categories = []string{
		"publicis paycheck",
		"rent",
		"parents",
		"other",
	}
}
func (t *templateData) DefaultExpenseCategories() {
	t.Categories = []string{
		"restaurant",
		"groceries",
		"home",
		"cat",
		"bills",
		"gym",
		"daki",
		"clothes",
		"health",
		"luxury",
		"other",
	}
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("02 Jan 2006 at 15:04")
}

var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
	pages, err := fs.Glob(ui.Files, "html/pages/*tmpl.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// If you do not use embedded fieles:
		// ts, err := template.New(name).Funcs(functions).ParseFiles("./ui/html/base.tmpl.html")
		// if err != nil {
		// 	return nil, err
		// }
		//
		// ts, err = ts.ParseGlob("./ui/html/partials/nav.tmpl.html")
		// if err != nil {
		// 	return nil, err
		// }
		//
		// ts, err = ts.ParseFiles(page)
		// if err != nil {
		// 	return nil, err
		// }

		patterns := []string{
			"html/base.tmpl.html",
			"html/partials/*.tmpl.html",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
