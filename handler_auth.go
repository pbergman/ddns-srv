package main

import (
	"net/http"
	"net/url"
)

func NewAuthHandler(users *UserList) Handler {
	return &AuthenticationHandler{
		users: users,
	}
}

type AuthenticationHandler struct {
	users *UserList
}

func (u *AuthenticationHandler) Supports(_ *url.URL) bool {
	return true
}

func (u *AuthenticationHandler) Handle(response http.ResponseWriter, request *http.Request) HandleResult {

	user, passwd, ok := request.BasicAuth()

	if false == ok || false == u.users.Authenticate(user, passwd) {
		response.Header().Add("WWW-Authenticate", `Basic realm="DDNS Server"`)
		http.Error(response, "", http.StatusUnauthorized)
		return StopPropagation
	}

	return ContinuePropagation
}
