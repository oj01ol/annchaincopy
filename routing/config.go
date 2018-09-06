package routing

import (
	"time"
)

type tbConfig struct {
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

func (c *tbConfig) SetDefault() {
	c.alpha = 3
	c.findsize = 16
	c.bucketSize = 16
	c.maxFindFailures = 5
	c.seedCount = 30
	c.seedMaxAge = 7 * 24 * time.Hour
	c.maxReplacements = 10
	c.refreshInterval = 30 * time.Second
	c.revalidateInterval = 30 * time.Second
	c.updateHashLength(40)
}

func (c *tbConfig) updateHashLength(v int) {
	c.HashLength = v
	c.hashBits = c.HashLength * 8
	c.nBuckets = c.hashBits / 15                  //10 21
	c.bucketMinDistance = c.hashBits - c.nBuckets //0~151,152,..,160
}

type dbConfig struct {
	nodeDBNilHash        Hash          // Special node ID to use as a nil element.
	nodeDBNodeExpiration time.Duration // Time after which an unseen node should be dropped.
	nodeDBCleanupCycle   time.Duration // Time period for running the expiration task.

	nodeDBItemPrefix []byte // Identifier to prefix node entries

	nodeDBDiscoverRoot      string
	nodeDBDiscoverPing      string
	nodeDBDiscoverPong      string
	nodeDBDiscoverFindFails string
}

func (c *dbConfig) SetDefault() {
	c.nodeDBNilHash = NewHash()             // Special node ID to use as a nil element.
	c.nodeDBNodeExpiration = 24 * time.Hour // Time after which an unseen node should be dropped.
	c.nodeDBCleanupCycle = time.Hour        // Time period for running the expiration task.
	c.nodeDBItemPrefix = []byte("n:")       // Identifier to prefix node entries
	c.nodeDBDiscoverRoot = ":discover"
	c.nodeDBDiscoverPing = c.nodeDBDiscoverRoot + ":lastping"
	c.nodeDBDiscoverPong = c.nodeDBDiscoverRoot + ":lastpong"
	c.nodeDBDiscoverFindFails = c.nodeDBDiscoverRoot + ":findfail"
}

type Configurable struct {
	HashLength int
}

func NewConfigurable() *Configurable {
	return &Configurable{}
}
func NewCgWithParam(hl int) *Configurable {
	return &Configurable{
		HashLength: hl,
	}
}

var c *tbConfig
var dbc *dbConfig

// Init method initializes params used by the module,
// so call it before using any other methods in this package.
// Or it will call panic
func Init(cg *Configurable) {
	initConfig(cg)
}

func initConfig(cg *Configurable) {
	if cg == nil {
		cg = NewConfigurable()
	}
	if c == nil {
		c = &tbConfig{}
		c.SetDefault()
	}
	if cg.HashLength > 0 {
		c.updateHashLength(cg.HashLength)
	}
	if dbc == nil {
		dbc = &dbConfig{}
		dbc.SetDefault()
	}
	return

}
