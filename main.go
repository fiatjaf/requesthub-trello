package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/graphql-go/graphql"
	"github.com/jingweno/jqplay/jq"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/lucsky/cuid"
	"github.com/rs/zerolog"
	"gopkg.in/redis.v5"
	"gopkg.in/tylerb/graceful.v1"
)

type Settings struct {
	ServiceURL   string `envconfig:"SERVICE_URL" required:"true"`
	Port         string `envconfig:"PORT" required:"true"`
	JQPath       string `envconfig:"JQ_PATH" required:"true"`
	DatabaseURL  string `envconfig:"DATABASE_URL" required:"true"`
	RedisURL     string `envconfig:"REDIS_URL"`
	TrelloAPIKey string `envconfig:"TRELLO_API_KEY" required:"true"`
	HashidsSalt  string `envconfig:"HASHIDS_SALT" required:"true"`
}

var err error
var pg *sqlx.DB
var rds *redis.Client
var s Settings
var router *mux.Router
var schema graphql.Schema
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	pg, err = sqlx.Connect("postgres", s.DatabaseURL)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("error connecting to postgres")
	}

	rurl, err := url.Parse(s.RedisURL)
	if s.RedisURL != "" && err == nil {
		passw, _ := rurl.User.Password()
		rds = redis.NewClient(&redis.Options{
			Addr:     rurl.Host,
			Password: passw,
		})
	} else {
		log.Debug().Str("url", s.RedisURL).
			Msg("invalid redis url, request logging capabilities will not work")
	}

	jq.Path = s.JQPath

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	router = mux.NewRouter()

	// set request ids for everybody
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), "request-id", cuid.Slug()))
			next.ServeHTTP(w, r)
		})
	})

	router.Path("/trello/card").Methods("GET").HandlerFunc(GetCard)
	router.Path("/trello/card").Methods("PUT").HandlerFunc(SetCard)
	router.Path("/trello/card").Methods("DELETE").HandlerFunc(DelCardEndpoint)

	// receive webhooks here
	router.HandleFunc("/w/{address}", func(w http.ResponseWriter, r *http.Request) {
		log := log.With().Str("req", r.Context().Value("request-id").(string)).Logger()

		var targets []EndpointTarget
		var address = mux.Vars(r)["address"]
		var errs = make(chan error, len(targets))
		var sbody string
		var c = 0

		var errorCode = 400
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			goto end
		}

		if !json.Valid(body) {
			err = errors.New("got invalid json")
			goto end
		}

		// save last requests for this endpoint on redis
		rds.LPush(address+":lreqs", body)
		rds.Expire(address+":lreqs", time.Hour*24*30)
		if rand.Intn(7) == 1 {
			rds.LTrim(address+"lreqs", 0, 5)
		}

		targets, err = getEndpointTargets(pg, address)
		if err != nil {
			errorCode = 500
			if err == sql.ErrNoRows {
				// try to return some http codes that will potentially cancel the webhook
				// on the server that is sending it
				if rand.Intn(7) == 1 {
					errorCode = 410
					// at least trello will cancel the webhook on a 410
				} else {
					errorCode = 200
					// sometimes webhooks will try to resend if you return an error code
					// to it's better to just return 200 most of the times
				}
			}
			goto end
		}

		sbody = string(body)
		if len(sbody) > 80 {
			sbody = sbody[:50]
		}

		log.Debug().Str("addr", address).Str("body", sbody).Msg("got webhook")

		for _, target := range targets {
			go func(target EndpointTarget) {
				log := log.With().Str("filter", target.Filter).Logger()

				ev := &jq.JQ{
					string(body),
					target.Filter,
					[]jq.JQOpt{
						{"compact-output", true},
						{"raw-output", true},
					},
				}

				evctx, evcancel := context.WithTimeout(r.Context(), time.Second*2)
				defer evcancel()
				value := &bytes.Buffer{}
				ev.Eval(evctx, value)

				reqctx, reqcancel := context.WithTimeout(r.Context(), time.Second*10)
				defer reqcancel()
				req, err := http.NewRequest("", "", &bytes.Buffer{})
				if err != nil {
					panic(err)
				}

				log.Debug().Str("data", value.String()).Msg("dispatching")
				errs <- dispatch(
					req.WithContext(reqctx),
					value.Bytes(),
					target,
				)
			}(target)
		}

		err = nil
		c = 0
		for terr := range errs {
			if err != nil {
				log.Warn().Err(terr).Msg("dispatch error")
				err = terr
			}

			c++
			if c >= len(targets) {
				close(errs)
			}
		}

	end:
		if err != nil {
			log.Warn().Err(err).Str("body", sbody).Msg("error handling webhook")
			http.Error(w, "error handling webhook", errorCode)
			return
		}
		w.WriteHeader(200)
	})

	router.Path("/favicon.ico").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./icon.png")
			return
		})

	router.PathPrefix("/powerup/").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")

			if r.URL.Path[len(r.URL.Path)-5:] == ".html" {
				http.ServeFile(w, r, "./powerup/basic.html")
				return
			}

			if r.URL.Path == "/powerup/icon.svg" {
				color := "#" + r.URL.Query().Get("color")
				secondary := "#dd5415"
				if color == "#" {
					color = "#999"
				}
				if strings.HasPrefix(color, "#999") {
					secondary = "#999"
				}

				w.Header().Set("Content-Type", "image/svg+xml")
				fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:sketch="http://www.bohemiancoding.com/sketch/ns" viewBox="0 0 100 125" version="1.1" x="0px" y="0px"><g stroke="none" stroke-width="1" fill="null" fill-rule="evenodd" sketch:type="MSPage"><g sketch:type="MSArtboardGroup" fill="%s"><path d="M76.4951181,73.9145361 L29.7416046,73.9145361 L29.7416046,78.9216292 C28.8333748,84.0543287 24.331884,87.9538209 18.915493,87.9538209 C12.8441011,87.9538209 7.92226559,83.0541766 7.92226559,77.0101589 C7.92226559,73.2805892 9.7963671,69.9867658 12.658225,68.0110376 L9.50582555,63.5279145 L9.50582555,63.5279145 C5.22643184,66.4933182 2.4256519,71.4261347 2.4256519,77.0101589 C2.4256519,86.0761854 9.80840521,93.4256519 18.915493,93.4256519 C27.0826867,93.4256519 33.8630986,87.5149032 35.1756484,79.7560854 L35.1756484,79.7560854 L76.2727359,79.7560854 C77.2245493,81.3860971 78.9977998,82.4819899 81.028169,82.4819899 C84.0638649,82.4819899 86.5247827,80.0321677 86.5247827,77.0101589 C86.5247827,73.9881501 84.0638649,71.538328 81.028169,71.538328 C79.146664,71.538328 77.4859566,72.4794128 76.4951181,73.9145361 Z" sketch:type="MSShapeGroup"/><path d="M13.9970435,74.7331616 C14.0920108,74.4983716 14.2044037,74.2673509 14.3346608,74.041739 C15.3445598,72.2925426 17.1729461,71.3107523 19.055871,71.3013474 L39.6682879,35.599594 L39.6682879,35.599594 C33.5944678,30.5874599 31.8490151,21.7752971 35.919978,14.7241825 C40.4574517,6.86504756 50.5025335,4.16979473 58.3562962,8.70416677 C63.1781003,11.4880366 66.0529051,16.3497657 66.5038811,21.5073647 L66.5038811,21.5073647 L61.0496311,21.9855501 C60.749694,18.546218 58.833007,15.3039392 55.6176886,13.4475743 C50.3818468,10.4246596 43.6851256,12.2214948 40.6601432,17.4609181 C37.6977221,22.5919819 39.3553436,29.1218655 44.342833,32.2289783 L47.3568902,33.969145 L24.0107416,74.4058605 L24.0107416,74.4058605 C24.6741594,75.7914555 24.7457069,77.4352393 24.1133132,78.9183847 C23.3416855,81.0070401 21.3316098,82.496256 18.9734712,82.496256 C15.9484888,82.496256 13.496256,80.0456995 13.496256,77.0227848 C13.496256,76.2053441 13.6755734,75.4297567 13.9970435,74.7331616 Z" sketch:type="MSShapeGroup"/><path fill="%s" d="M49.662923,28.3537714 L72.775405,68.3857646 L75.5278906,66.7966163 L75.5745661,66.8774606 C75.5972847,66.8641312 75.6200684,66.8508704 75.6429171,66.8376787 C80.8823981,63.8126629 87.5575782,65.5653749 90.5523492,70.7524704 C93.5471202,75.939566 91.7274232,82.5967975 86.4879423,85.6218133 C83.2579165,87.4866695 79.48226,87.5357725 76.359624,86.0838539 L74.0282197,91.020885 L74.0282197,91.020885 C78.7079973,93.1885541 84.3615702,93.1108529 89.1991985,90.3178469 C97.05842,85.7803233 99.7879655,75.794476 95.295809,68.0138327 C91.2398724,60.9887444 82.6834446,58.1633014 75.2751126,61.0206617 L75.2751126,61.0206617 L54.7672927,25.5000756 C55.6506759,23.8891995 55.7003697,21.8799727 54.7216499,20.1847802 C53.2242644,17.5912325 49.8866743,16.7148765 47.2669338,18.2273844 C44.6471933,19.7398922 43.7373448,23.068508 45.2347303,25.6620558 C46.1863907,27.3103798 47.8813487,28.2650936 49.662923,28.3537714 Z" sketch:type="MSShapeGroup"/></g></g></svg>`, color, secondary)
				return
			}

			http.ServeFile(w, r, "."+r.URL.Path)
		},
	)

	// start the server
	log.Info().Str("port", os.Getenv("PORT")).Msg("listening.")
	graceful.Run(":"+os.Getenv("PORT"), 10*time.Second, router)
}
