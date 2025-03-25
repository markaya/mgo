package main

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
const authenticatedUser = contextKey("authenticatedUser")
