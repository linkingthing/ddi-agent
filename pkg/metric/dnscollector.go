package collector

import (
	"sort"
	"strconv"

	"github.com/linkingthing/ddi-agent/pkg/boltdb"
)

type DNSCollector struct {
	db *boltdb.BoltHandler
}

func newDNSCollector(db *boltdb.BoltHandler) *DNSCollector {
	return &DNSCollector{db: db}
}

func (c *DNSCollector) GetQPS(table string) (float64, error) {
	if kvs, timestamps, err := getKVsAndTimestampsFromDB(c.dbHandler, table); err != nil {
		return 0, err
	} else if len(kvs) > 1 {
		numPrev, err := strconv.Atoi(timestamps[len(timestamps)-2])
		if err != nil {
			return 0, err
		}

		numLast, err := strconv.Atoi(timestamps[len(timestamps)-1])
		if err != nil {
			return 0, err
		}

		queryPrev, err := strconv.Atoi(string(kvs[timestamps[len(timestamps)-2]]))
		if err != nil {
			return 0, err
		}

		queryLast, err := strconv.Atoi(string(kvs[timestamps[len(timestamps)-1]]))
		if err != nil {
			return 0, err
		}

		if queryLast-queryPrev > 0 && numLast-numPrev > 0 {
			return float64(queryLast-queryPrev) / float64(numLast-numPrev), nil
		}
	}

	return 0, nil
}

func getKVsAndTimestampsFromDB(db *boltdb.BoltHandler, table string) (map[string][]byte, []string, error) {
	kvs, err := db.TableKVs(table)
	if err != nil {
		return nil, nil, err
	}

	var timestamps []string
	for k, _ := range kvs {
		timestamps = append(timestamps, k)
	}

	sort.Strings(timestamps)
	return kvs, timestamps, nil
}

func (c *DNSCollector) GetQueries(table string) (float64, error) {
	var query int
	kvs, timestamps, err := getKVsAndTimestampsFromDB(c.dbHandler, table)
	if err != nil {
		return 0, err
	} else if len(kvs) > 1 {
		query, err = strconv.Atoi(string(kvs[timestamps[len(timestamps)-1]]))
	}

	return float64(query), err
}
