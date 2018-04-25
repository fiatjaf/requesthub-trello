package main

import (
	"errors"
	"net/http"
)

func dispatch(req *http.Request, value []byte, t EndpointTarget) error {
	if t.Kind == "trello:comment" {
		return dispatchTrello(req, value, t)
	}
	return errors.New("unrecognized output kind '" + t.Kind + "' for target '" + t.Target + "'")
}
