package connector

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type Connector struct {
	DBHostname     string
	DBName         string
	DBUser         string
	DBUserPassword string
	DBMaxConns     int
}

func (c Connector) OpenAndCreateDB() (*sqlx.DB, error) {
	connConfig := pgx.ConnConfig{
		Host:     c.DBHostname,
		Database: c.DBName,
		User:     c.DBUser,
		Password: c.DBUserPassword,
	}
	connPool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		AfterConnect:   nil,
		MaxConnections: c.DBMaxConns,
		AcquireTimeout: 30 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	nativeDB := stdlib.OpenDBFromPool(connPool)

	log.Println("OpenAndCreateDB: the connection was created")
	db_conn := sqlx.NewDb(nativeDB, "pgx")

	if db_conn == nil {
		return nil, fmt.Errorf("Error")
	}

	if query, err := ioutil.ReadFile("db/repeater.sql"); err != nil {
		return nil, err
	} else {
		if _, err := db_conn.Exec(string(query)); err != nil {
			return nil, err
		}
	}

	return db_conn, err
}

func (c Connector) OpenDB() (*sqlx.DB, error) {
	connConfig := pgx.ConnConfig{
		Host:     c.DBHostname,
		Database: c.DBName,
		User:     c.DBUser,
		Password: c.DBUserPassword,
	}
	connPool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		AfterConnect:   nil,
		MaxConnections: c.DBMaxConns,
		AcquireTimeout: 30 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	nativeDB := stdlib.OpenDBFromPool(connPool)

	log.Println("OpenDB: the connection was created")
	return sqlx.NewDb(nativeDB, "pgx"), nil
}
