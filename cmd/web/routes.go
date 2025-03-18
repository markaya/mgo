package main

import (
	"net/http"

	"github.com/markaya/meinappf/ui"
)

func (app *application) routes() http.Handler {

	mux := http.NewServeMux()

	// TODO: Find a way to use it only on handfull of requests, not all of them
	dynamic := func(handler http.Handler) http.Handler {
		return app.sessionManager.LoadAndSave(app.authenticate(handler))
	}

	// NOTE: Protected routes
	protected := func(handler http.Handler) http.Handler {
		return dynamic(app.requireAuthentication(handler))
	}

	fileServer := http.FileServer(http.FS(ui.Files))

	// NOTE: When using embeded files we do not need to strip prefix
	mux.Handle("GET /static/{filepath...}", fileServer)

	/*
		NOTE: There is one last bit of syntax. As we showed above, patterns ending in a slash,
		like /posts/, match all paths beginning with that string.
		To match only the path with the trailing slash, you can write /posts/{$}.
		That will match /posts/ but not /posts or /posts/234.
	*/

	// NOTE: For test purpose
	mux.HandleFunc("GET /ping", app.ping)

	// NOTE: Regular Session
	mux.Handle("GET /{$}", dynamic(http.HandlerFunc(app.home)))
	mux.Handle("GET /about", dynamic(http.HandlerFunc(app.aboutView)))

	// NOTE: User Paths
	mux.Handle("GET /user/signup", dynamic(http.HandlerFunc(app.userSignup)))
	mux.Handle("POST /user/signup", dynamic(http.HandlerFunc(app.userSignupPost)))
	mux.Handle("GET /user/login", dynamic(http.HandlerFunc(app.userLogin)))
	mux.Handle("POST /user/login", dynamic(http.HandlerFunc(app.userLoginPost)))

	// NOTE: Auth Session
	mux.Handle("POST /user/logout", protected(dynamic(http.HandlerFunc(app.userLogoutPost))))
	mux.Handle("GET /user/view/", protected(dynamic(http.HandlerFunc(app.userView))))
	mux.Handle("GET /user/password/update", protected(dynamic(http.HandlerFunc(app.accountPasswordUpdate))))
	mux.Handle("POST /user/password/update", protected(dynamic(http.HandlerFunc(app.accountPasswordUpdatePost))))

	// NOTE: Accounts
	mux.Handle("GET /accounts/", protected(dynamic(http.HandlerFunc(app.accountsView))))
	mux.Handle("GET /account/view/{id}", protected(dynamic(http.HandlerFunc(app.accountView))))
	mux.Handle("GET /account/create", protected(dynamic(http.HandlerFunc(app.accountCreate))))
	mux.Handle("POST /account/create", protected(dynamic(http.HandlerFunc(app.accountCreatePost))))
	mux.Handle("GET /account/rebalance/{id}", protected(dynamic(http.HandlerFunc(app.accountRebalanceView))))
	mux.Handle("POST /account/rebalance/", protected(dynamic(http.HandlerFunc(app.accountRebalancePost))))

	// NOTE: Transactions
	mux.Handle("GET /transactions/", protected(dynamic(http.HandlerFunc(app.transactionsView))))
	mux.Handle("GET /transaction/create/{ttype}", protected(dynamic(http.HandlerFunc(app.transactionCreate))))
	mux.Handle("POST /transaction/create/{$}", protected(dynamic(http.HandlerFunc(app.transactionCreatePost))))

	// NOTE: Transfers
	mux.Handle("GET /transfers/", protected(dynamic(http.HandlerFunc(app.transfersView))))
	mux.Handle("GET /trnasfers/view/{id}", protected(dynamic(http.HandlerFunc(app.transferView))))
	mux.Handle("GET /transfer/create/", protected(dynamic(http.HandlerFunc(app.transferCreate))))
	mux.Handle("POST /transfer/create/", protected(dynamic(http.HandlerFunc(app.transferCreatePost))))

	// Match everything else
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	// NOTE: Middleware
	// [IN] (Log request) -> (Add Headers) -> (Serve mux)
	// [OUT] (Recover Panic)    <-			  (Serve mux)
	return app.recoverPanic(app.logRequest(secureHeaders(mux)))
}
