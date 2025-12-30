package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/recommendations", app.listRecommendationsHandler)
	router.HandlerFunc(http.MethodPost, "/v1/recommendations", app.requirePermission("recommendations:write", app.createRecommendationHandler))
	router.HandlerFunc(http.MethodGet, "/v1/recommendations/:id", app.requirePermission("recommendations:read", app.showRecommendationHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/recommendations/:id", app.requirePermission("recommendations:write", app.updateRecommendationHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/recommendations/:id", app.requirePermission("recommendations:write", app.deleteRecommendationHandler))

	router.HandlerFunc(http.MethodPost, "/v1/comments", app.requirePermission("comments:write", app.createCommentHandler))

	router.HandlerFunc(http.MethodGet, "/v1/search", app.requirePermission("recommendations:write", app.searchMusicData))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodGet, "/v1/users/:username", app.showUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	router.Handler(http.MethodGet, "/debug/vars/", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
