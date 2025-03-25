package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/markaya/meinappf/internal/models"
	"github.com/markaya/meinappf/internal/services"
	"github.com/markaya/meinappf/ui"
)

// TODO: Add success and fail flash.
type templateData struct {
	CurrentYear         int
	DateStringNow       string
	Form                any
	Flash               string
	IsAuthenticated     bool
	User                *models.User
	Account             *models.Account
	Accounts            []*models.Account
	Categories          []string
	UserTotalReport     services.TotalReport
	DateFilter          map[string]time.Time
	IncomeTransactions  []*models.Transaction
	ExpenseTransactions []*models.Transaction
}

func (t *templateData) WithDefaultDateFilter() {
	filterMap := make(map[string]time.Time)
	now := time.Now()
	// Default to the first and last day of the current month if dates are not provided
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 999999999, time.UTC)
	filterMap["startDate"] = startDate
	filterMap["endDate"] = endDate
	t.DateFilter = filterMap
}

func (t *templateData) DefaultIncomeCategories() {
	t.Categories = []string{
		"publicis",
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
		"commute",
		"health",
		"luxury",
		"other",
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("02 Jan 2006 at 15:04")
}

func htmlDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

var functions = template.FuncMap{
	"humanDate":   humanDate,
	"htmlDate":    htmlDate,
	"formatFloat": formatFloat,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
	pages, err := fs.Glob(ui.Files, "html/pages/*.html")
	if err != nil {
		return nil, err
	}

	forms, err := fs.Glob(ui.Files, "html/forms/*.html")
	if err != nil {
		return nil, err
	}

	for _, form := range forms {
		name := filepath.Base(form)
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, form)
		if err != nil {
			return nil, err
		}
		cache[name] = ts
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/foundation.html",
			"html/partials/*.html",
			"html/forms/*.html",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		// Add to cache models that are not

		cache[name] = ts
	}

	return cache, nil
}

// func newTemplateCache() (map[string]*template.Template, error) {
// 	cache := map[string]*template.Template{}
//
// 	// pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
// 	pages, err := fs.Glob(ui.Files, "html/pages/*tmpl.html")
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	forms, err := fs.Glob(ui.Files, "html/forms/*tmpl.html")
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	for _, form := range forms {
// 		name := filepath.Base(form)
// 		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, form)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cache[name] = ts
// 	}
//
// 	for _, page := range pages {
// 		name := filepath.Base(page)
//
// 		// If you do not use embedded fieles:
// 		// ts, err := template.New(name).Funcs(functions).ParseFiles("./ui/html/base.tmpl.html")
// 		// if err != nil {
// 		// 	return nil, err
// 		// }
// 		//
// 		// ts, err = ts.ParseGlob("./ui/html/partials/nav.tmpl.html")
// 		// if err != nil {
// 		// 	return nil, err
// 		// }
// 		//
// 		// ts, err = ts.ParseFiles(page)
// 		// if err != nil {
// 		// 	return nil, err
// 		// }
// 			patterns := []string{
// 				"html/base.tmpl.html",
// 				"html/partials/*.tmpl.html",
// 				"html/forms/*.tmpl.html",
// 				page,
// 			}
// 		}
//
// 		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		// Add to cache models that are not
//
// 		cache[name] = ts
// 	}
//
// 	return cache, nil
// }
