package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/lucsky/cuid"
)

func GetCard(w http.ResponseWriter, r *http.Request) {
	card := r.URL.Query().Get("card")
	token := r.URL.Query().Get("token")

	// check if token is correct and card exists and whatever
	username, err := trelloUsernameAndCard(token, card)
	if err != nil {
		log.Warn().Err(err).
			Str("token", token).
			Str("card", card).
			Msg("failed to get member and card info")
		http.Error(w, err.Error(), 401)
		return
	}

	endpoints := []struct {
		Address string `json:"address" db:"address"`
		Filter  string `json:"filter" db:"filter"`
	}{}
	err = pg.Select(&endpoints, `
SELECT address, filter
FROM pipe
INNER JOIN input ON pipe.i = input.address
INNER JOIN output ON pipe.o = output.id
WHERE output.target = $1
  AND output.owner = $2
    `, card, username+"@trello")
	if err != nil &&
		err != sql.ErrNoRows /* now rows is a valid result */ {

		log.Warn().Err(err).Str("card", card).
			Msg("failed to fetch endpoint data for card")
		http.Error(w, "failed to fetch endpoint data for card: "+err.Error(), 500)
	}

	json.NewEncoder(w).Encode(endpoints)
}

func SetCard(w http.ResponseWriter, r *http.Request) {
	var info struct {
		Address    string `json:"address"`
		NewAddress string `json:"newAddress"`
		Filter     string `json:"filter"`
		Card       string `json:"card"`
		Token      string `json:"token"`
	}
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// check if token is correct and card exists and whatever
	username, err := trelloUsernameAndCard(info.Token, info.Card)
	if err != nil {
		log.Warn().Err(err).
			Str("token", info.Token).
			Str("card", info.Card).
			Msg("given token doesn't have access to card")
		http.Error(w, err.Error(), 401)
		return
	}

	jsonData, _ := json.Marshal(map[string]string{"token": info.Token})

	if info.Address == "" {
		// means a new endpoint must be created

		if info.NewAddress == "" {
			// people can choose their addresses, but we provide defaults
			info.NewAddress = cuid.Slug()
		}
		_, err = pg.Exec(`
WITH i AS (
  INSERT INTO input (address, owner)
  VALUES ($1, $2)
  RETURNING address
),
     o AS (
  INSERT INTO output (kind, target, filter, owner, data)
  VALUES ('trello:comment', $3, $4, $2, $5)
  RETURNING id
)
INSERT INTO pipe VALUES ((SELECT address FROM i), (SELECT id FROM o))
                `, info.NewAddress, username+"@trello", info.Card, info.Filter, jsonData)
	} else {
		if info.NewAddress == "" {
			// people can change their addresses, by default they stay the same
			info.NewAddress = info.Address
		}
		_, err = pg.Exec(`
WITH io AS (
  SELECT * FROM pipe WHERE i = $1
),
      i AS (
  UPDATE input SET address = $2
  WHERE address = (SELECT i FROM io)
),
      o AS (
  UPDATE output SET filter = $3, target = $4, data = $5
  WHERE id = (SELECT o FROM io)
)
SELECT NULL
                `, info.Address, info.NewAddress, info.Filter, info.Card, jsonData)
	}
	if err != nil {
		log.Warn().Err(err).Str("card", info.Card).
			Msg("failed to create or update endpoint.")
		http.Error(w, "failed to create or update endpoint: "+err.Error(), 403)
		return
	}

	json.NewEncoder(w).Encode(info.NewAddress)
}

func trelloUsernameAndCard(token string, card string) (username string, err error) {
	var user struct {
		Name string `json:"username"`
	}
	resp, err := http.Get("https://api.trello.com/1/members/me?key=" + s.TrelloAPIKey + "&token=" + token + "&fields=username")
	if err == nil && resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		err = errors.New("trello returned '" + string(b) + "' on /members call.")
	}
	if err != nil {
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return
	}
	username = user.Name

	resp, err = http.Get("https://api.trello.com/1/cards/" + card + "?key=" + s.TrelloAPIKey + "&token=" + token + "&fields=shortLink")
	if err == nil && resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		err = errors.New("trello returned '" + string(b) + "' on /cards call.")
	}
	if err != nil {
		return
	}

	return
}

func dispatchTrello(req *http.Request, value []byte, target EndpointTarget) error {
	var t struct {
		Token string `json:"token"`
	}
	err = target.Data.Unmarshal(&t)
	if err != nil {
		log.Error().Err(err).Str("json", target.Data.String()).
			Msg("failed to parse trello token")
		return err
	}

	params := url.Values{}
	params.Add("key", s.TrelloAPIKey)
	params.Add("token", t.Token)
	params.Add("text", "Got webhook on **"+target.Address+"**:\n\n>"+
		strings.Join(strings.Split(string(value), "\n"), "\n>"))

	req.Method = "POST"
	req.URL.Scheme = "https"
	req.URL.Host = "api.trello.com"
	req.URL.Path = "/1/cards/" + target.Target + "/actions/comments"
	req.URL.RawQuery = params.Encode()

	resp, err := http.DefaultClient.Do(req)

	if err == nil && resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		err = errors.New("trello returned '" + string(b) + "' on /comments call.")
	}
	if err != nil {
		return err
	}
	return nil
}
