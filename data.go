package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

type EndpointTarget struct {
	Address string         `db:"address"`
	Filter  string         `db:"filter"`
	Kind    string         `db:"kind"`
	Target  string         `db:"target"`
	Data    types.JSONText `db:"data"`
}

func getEndpointTargets(pg *sqlx.DB, address string) (targets []EndpointTarget, err error) {
	err = pg.Select(&targets, `
SELECT address, filter, kind, target, data 
FROM pipe
INNER JOIN input ON pipe.i = input.address
INNER JOIN output ON pipe.o = output.id
WHERE input.address = $1
    `, address)
	return
}
