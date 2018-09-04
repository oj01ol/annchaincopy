package routing

import (
	"time"
)

type Config struct {
	alpha              int
	HashLength         int
	hashBits           int
	findsize           int
	bucketSize         int
	maxFindFailures    int
	nBuckets           int //10 21
	seedCount          int
	seedMaxAge         time.Duration
	maxReplacements    int
	bucketMinDistance  int //0~151,152,..,160
	refreshInterval    time.Duration
	revalidateInterval time.Duration
}

func (c *Config) SetDefault() {
	c.alpha = 3
	c.HashLength = 40
	c.hashBits = c.HashLength * 8
	c.findsize = 16
	c.bucketSize = 16
	c.maxFindFailures = 5
	c.nBuckets = c.hashBits / 15 //10 21
	c.seedCount = 30
	c.seedMaxAge = 7 * 24 * time.Hour
	c.maxReplacements = 10
	c.bucketMinDistance = c.hashBits - c.nBuckets //0~151,152,..,160
	c.refreshInterval = 30 * time.Second
	c.revalidateInterval = 30 * time.Second
}

var c *Config
var dbc *DBConfig

func init() {
	initConfig()
}

func initConfig() {
	if c == nil {
		c = &Config{}
		c.SetDefault()
	}

	if dbc == nil {
		dbc = &DBConfig{}
		dbc.SetDefault()
	}
}

type DBConfig struct {
	nodeDBNilHash        Hash          // Special node ID to use as a nil element.
	nodeDBNodeExpiration time.Duration // Time after which an unseen node should be dropped.
	nodeDBCleanupCycle   time.Duration // Time period for running the expiration task.

	nodeDBItemPrefix []byte // Identifier to prefix node entries

	nodeDBDiscoverRoot      string
	nodeDBDiscoverPing      string
	nodeDBDiscoverPong      string
	nodeDBDiscoverFindFails string
}

func (c *DBConfig) SetDefault() {
	c.nodeDBNilHash = NewHash()             // Special node ID to use as a nil element.
	c.nodeDBNodeExpiration = 24 * time.Hour // Time after which an unseen node should be dropped.
	c.nodeDBCleanupCycle = time.Hour        // Time period for running the expiration task.
	c.nodeDBItemPrefix = []byte("n:")       // Identifier to prefix node entries
	c.nodeDBDiscoverRoot = ":discover"
	c.nodeDBDiscoverPing = c.nodeDBDiscoverRoot + ":lastping"
	c.nodeDBDiscoverPong = c.nodeDBDiscoverRoot + ":lastpong"
	c.nodeDBDiscoverFindFails = c.nodeDBDiscoverRoot + ":findfail"
}
