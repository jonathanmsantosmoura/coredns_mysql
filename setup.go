package coredns_mysql

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("mysql")

const (
	defaultTtl                = 360
	defaultMaxLifeTime        = 1 * time.Minute
	defaultMaxOpenConnections = 10
	defaultMaxIdleConnections = 10
	defaultZoneUpdateTime     = 10 * time.Minute
)

func init() {
	caddy.RegisterPlugin("mysql", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	r, err := mysqlParse(c)
	if err != nil {
		return plugin.Error("mysql", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		r.Next = next
		return r
	})

	return nil
}

func mysqlParse(c *caddy.Controller) (*CoreDNSMySql, error) {
	mysql := CoreDNSMySql{
		TablePrefix: "coredns_",
		Ttl:         300,
	}
	var err error

	c.Next()
	if c.NextBlock() {
		for {
			switch c.Val() {
			case "dsn":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				mysql.Dsn = c.Val()
			case "table_prefix":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				mysql.TablePrefix = c.Val()
			case "max_lifetime":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val time.Duration
				val, err = time.ParseDuration(c.Val())
				if err != nil {
					val = defaultMaxLifeTime
				}
				mysql.MaxLifetime = val
			case "max_open_connections":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultMaxOpenConnections
				}
				mysql.MaxOpenConnections = val
			case "max_idle_connections":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultMaxIdleConnections
				}
				mysql.MaxIdleConnections = val
			case "zone_update_interval":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val time.Duration
				val, err = time.ParseDuration(c.Val())
				if err != nil {
					val = defaultZoneUpdateTime
				}
				mysql.zoneUpdateTime = val
			case "ttl":
				if !c.NextArg() {
					return &CoreDNSMySql{}, c.ArgErr()
				}
				var val int
				val, err = strconv.Atoi(c.Val())
				if err != nil {
					val = defaultTtl
				}
				mysql.Ttl = uint32(val)
			default:
				if c.Val() != "}" {
					return &CoreDNSMySql{}, c.Errf("unknown property '%s'", c.Val())
				}
			}

			if !c.Next() {
				break
			}
		}

	}

	db, err := mysql.initDB()
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	mysql.tableName = mysql.TablePrefix + "records"

	return &mysql, nil
}

var databaseCoredns *sql.DB

func (handler *CoreDNSMySql) initDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", os.ExpandEnv(handler.Dsn))
	if err != nil {
		log.Info(err)
		return nil, err
	}

	// db.SetConnMaxLifetime(handler.MaxLifetime)
	db.SetMaxOpenConns(handler.MaxOpenConnections)
	db.SetMaxIdleConns(handler.MaxIdleConnections)

	databaseCoredns = db

	log.Info("MaxOpenConnections:" + fmt.Sprint(handler.MaxOpenConnections))
	log.Info("MaxIdleConnections:" + fmt.Sprint(handler.MaxIdleConnections))

	return db, nil
}
