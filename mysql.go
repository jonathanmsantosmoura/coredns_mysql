package coredns_mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func (handler *CoreDNSMySql) findRecord(zone string, name string, types ...string) ([]*Record, error) {
	// db, err := handler.db()
	// if err != nil {
	// 	return nil, err
	// }
	// defer db.Close()

	var query string
	if name == "" && types[0] == "SOA" {
		query = zone
	} else if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}

	sqlQuery := fmt.Sprintf("SELECT name, zone, ttl, record_type, content FROM %s WHERE zone = ? AND name = ? AND record_type IN ('%s')",
		handler.tableName,
		strings.Join(types, "','"))
	result, err := databaseCoredns.Query(sqlQuery, zone, query)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	var recordName string
	var recordZone string
	var recordType string
	var ttl uint32
	var content string
	records := make([]*Record, 0)
	for result.Next() {
		err = result.Scan(&recordName, &recordZone, &ttl, &recordType, &content)
		if err != nil {
			return nil, err
		}

		records = append(records, &Record{
			Name:       recordName,
			Zone:       recordZone,
			RecordType: recordType,
			Ttl:        ttl,
			Content:    content,
			handler:    handler,
		})
	}

	result.Close()

	// If no records found, check for wildcard records.
	// if len(records) == 0 && name != zone {
	// 	return handler.findWildcardRecords(zone, name, types...)
	// }

	return records, nil
}

func (handler *CoreDNSMySql) findRecordByZoneAndNameStatic(zone string, name string) ([]*Record, error) {
	// db, err := handler.db()
	// if err != nil {
	// 	return nil, err
	// }
	// defer db.Close()

	var recordName = name
	var recordZone = zone
	var recordType string
	var ttl uint32 = 60
	var content string = fmt.Sprintf("{'ttl':100,'mbox':hostmaster.%s,'ns':ns1.%s, 'refresh':44,'retry':55,'expire':66}", zone, zone)

	records := make([]*Record, 0)

	records = append(records, &Record{
		Name:       recordName,
		Zone:       recordZone,
		RecordType: recordType,
		Ttl:        ttl,
		Content:    content,
		handler:    handler,
	})

	// If no records found, check for wildcard records.
	// if len(records) == 0 && name != zone {
	// 	return handler.findWildcardRecords(zone, name, types...)
	// }

	return records, nil
}

func (handler *CoreDNSMySql) findRecordByZoneAndName(zone string, name string) ([]*Record, error) {
	// db, err := handler.db()
	// if err != nil {
	// 	return nil, err
	// }
	// defer db.Close()

	var query string
	if name != zone {
		query = strings.TrimSuffix(name, "."+zone)
	}

	sqlQuery := fmt.Sprintf("SELECT name, zone, ttl, record_type, content FROM %s WHERE zone = ? AND name = ?",
		handler.tableName)
	result, err := databaseCoredns.Query(sqlQuery, zone, query)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	var recordName string
	var recordZone string
	var recordType string
	var ttl uint32
	var content string
	records := make([]*Record, 0)
	for result.Next() {
		err = result.Scan(&recordName, &recordZone, &ttl, &recordType, &content)
		if err != nil {
			return nil, err
		}

		records = append(records, &Record{
			Name:       recordName,
			Zone:       recordZone,
			RecordType: recordType,
			Ttl:        ttl,
			Content:    content,
			handler:    handler,
		})
	}

	result.Close()

	// If no records found, check for wildcard records.
	// if len(records) == 0 && name != zone {
	// 	return handler.findWildcardRecords(zone, name, types...)
	// }

	return records, nil
}

// findWildcardRecords attempts to find wildcard records
// recursively until it finds matching records.
// e.g. x.y.z -> *.y.z -> *.z -> *
// func (handler *CoreDNSMySql) findWildcardRecords(zone string, name string, types ...string) ([]*Record, error) {
// 	const (
// 		wildcard       = "*"
// 		wildcardPrefix = wildcard + "."
// 	)

// 	if name == wildcard {
// 		return nil, nil
// 	}

// 	name = strings.TrimPrefix(name, wildcardPrefix)

// 	target := wildcard
// 	i, shot := dns.NextLabel(name, 0)
// 	if !shot {
// 		target = wildcardPrefix + name[i:]
// 	}

// 	return handler.findRecord(zone, target, types...)
// }

func (handler *CoreDNSMySql) loadZones() error {
	// db, err := handler.db()
	// if err != nil {
	// 	return err
	// }
	// defer db.Close()

	result, err := databaseCoredns.Query("SELECT DISTINCT zone FROM " + handler.tableName)
	if err != nil {
		log.Info(err)
		return err
	}

	var zone string
	zones := make([]string, 0)
	for result.Next() {
		err = result.Scan(&zone)
		if err != nil {
			return err
		}

		zones = append(zones, zone)
	}

	result.Close()

	handler.lastZoneUpdate = time.Now()
	handler.zones = zones

	return nil
}

func (handler *CoreDNSMySql) hosts(zone string, name string) ([]dns.RR, error) {
	recs, err := handler.findRecord(zone, name, "A", "AAAA", "CNAME")
	if err != nil {
		log.Info(err)
		return nil, err
	}

	answers := make([]dns.RR, 0)

	for _, rec := range recs {
		switch rec.RecordType {
		case "A":
			aRec, _, err := rec.AsARecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		case "AAAA":
			aRec, _, err := rec.AsAAAARecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		case "CNAME":
			aRec, _, err := rec.AsCNAMERecord()
			if err != nil {
				return nil, err
			}
			answers = append(answers, aRec)
		}
	}

	return answers, nil
}
