package db

import (
	"github.com/heartbeatsjp/happo-agent/util"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	// DB is leveldb.DB
	DB *leveldb.DB
	// MetricsMaxLifetimeSeconds is as variable name
	MetricsMaxLifetimeSeconds int64
	// MachineStateMaxLifetimeSeconds is as variable name
	MachineStateMaxLifetimeSeconds int64
)

func init() {
	MetricsMaxLifetimeSeconds = 7 * 86400      //default is 7 days
	MachineStateMaxLifetimeSeconds = 3 * 86400 //default is 3 days
}

// Open open leveldb file
func Open(dbfile string) {
	log := util.HappoAgentLogger()
	var err error
	DB, err = leveldb.OpenFile(dbfile, nil)
	if err != nil {
		if errors.IsCorrupted(err) {
			log.Errorf("detect corrupted manifest file in %s: %s\n", dbfile, err)
			log.Errorln("attempt recover for", dbfile)

			DB, err = leveldb.RecoverFile(dbfile, nil)
			if err != nil {
				log.Fatalln("recover failed: ", err)
			}

			log.Errorln("recover corrupted manifest file succeeded")
		} else {
			log.Fatalln(err)
		}
	}
}

// Close close leveldb file
func Close() {
	log := util.HappoAgentLogger()
	var err error
	err = DB.Close()
	if err != nil {
		log.Error(err)
	}
}
