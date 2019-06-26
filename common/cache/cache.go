package cache

import (
	"encoding/gob"
	"encoding/hex"
	"github.com/Dreamacro/clash/constant"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cache store element with a expired time
type Cache struct {
	*cache
}

type cache struct {
	mapping sync.Map
	janitor *janitor
	count   uint32
}

type element struct {
	Expired time.Time
	Payload interface{}
}

// Put element in Cache with its ttl
func (c *cache) Put(key interface{}, payload interface{}, ttl time.Duration) {
	c.mapping.Store(key, &element{
		Payload: payload,
		Expired: time.Now().Add(ttl),
	})
	c.count++
}

func (c *cache) Exist(key interface{}) bool {
	_, exist := c.mapping.Load(key)
	return exist
}

// Get element in Cache, and drop when it expired
func (c *cache) Get(key interface{}) interface{} {
	item, exist := c.mapping.Load(key)
	if !exist {
		return nil
	}
	elm := item.(*element)
	payload, err := hex.DecodeString(elm.Payload.(string))
	if err != nil {
		c.mapping.Delete(key)
		return nil
	}
	return payload
}

// GetWithExpire element in Cache with Expire Time
func (c *cache) GetWithExpire(key interface{}) (payload interface{}, expired time.Time) {
	item, exist := c.mapping.Load(key)
	if !exist {
		return
	}
	elm := item.(*element)
	payload, err := hex.DecodeString(elm.Payload.(string))
	if err != nil {
		c.mapping.Delete(key)
		return
	}
	return payload, elm.Expired
}

func (c *cache) cleanup() {
	c.mapping.Range(func(k, v interface{}) bool {
		key := k.(string)
		elm := v.(*element)
		if time.Since(elm.Expired).Hours() > 72 {
			c.mapping.Delete(key)
		}
		return true
	})

	_ = c.Save()
}

func (c *cache) Save() error {
	if c.count == 0 {
		return nil
	}
	f, err := os.Create(filepath.Join(constant.Path.HomeDir(), "dnscache"))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	res := make(map[string]string)

	c.mapping.Range(func(k, v interface{}) bool {
		key := k.(string)
		elm := v.(*element)
		expireTime := elm.Expired.Unix()

		res[key] = elm.Payload.(string) + "," + strconv.FormatInt(expireTime, 10)
		return true
	})

	err = enc.Encode(res)
	if err != nil {
		return err
	}

	c.count = 0
	return nil
}

func (c *cache) Reload() {
	f, err := os.Open(filepath.Join(constant.Path.HomeDir(), "dnscache"))
	if err != nil {
		return
	}
	defer f.Close()

	dec := gob.NewDecoder(f)
	items := make(map[string]string)

	err = dec.Decode(&items)
	if err != nil {
		return
	}

	for k, v := range items {
		s := strings.Split(v, ",")

		var expireTime = time.Now()
		if len(s) > 1 {
			i, err := strconv.ParseInt(s[1], 10, 64)
			if err != nil {
				continue
			}
			expireTime = time.Unix(i, 0)
		}

		c.Put(k, s[0], time.Until(expireTime))
	}

	return
}

type janitor struct {
	interval time.Duration
	stop     chan struct{}
}

func (j *janitor) process(c *cache) {
	ticker := time.NewTicker(j.interval)
	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *Cache) {
	c.janitor.stop <- struct{}{}
}

// New return *Cache
func New(interval time.Duration) *Cache {
	j := &janitor{
		interval: interval,
		stop:     make(chan struct{}),
	}
	c := &cache{janitor: j}
	go j.process(c)
	C := &Cache{c}
	runtime.SetFinalizer(C, stopJanitor)
	return C
}
