// Package flagsdk evaluates last-known configurations during server outages.
package flagsdk

import (
	"context"
	"encoding/json"
	"flagcontrol/projects/flagcontrol"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Config struct {
	Tenant      string   `json:"tenant"`
	Key         string   `json:"key"`
	Environment string   `json:"environment"`
	Version     int      `json:"version"`
	Enabled     bool     `json:"enabled"`
	Percentage  int      `json:"percentage"`
	Allow       []string `json:"allow_users"`
}
type Client struct {
	BaseURL, Token string
	mu             sync.RWMutex
	cache          map[string]Config
}

func (c *Client) Refresh(ctx context.Context, key, environment string) error {
	req, e := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/flags/"+url.PathEscape(key)+"/"+url.PathEscape(environment), nil)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, e := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("flag service status %d", resp.StatusCode)
	}
	var config Config
	if e = json.NewDecoder(resp.Body).Decode(&config); e != nil {
		return e
	}
	if config.Key != key || config.Environment != environment || config.Percentage < 0 || config.Percentage > 100 {
		return fmt.Errorf("invalid config")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = map[string]Config{}
	}
	c.cache[key+"\x00"+environment] = config
	return nil
}
func (c *Client) Evaluate(key, environment, user string, fallback bool) bool {
	c.mu.RLock()
	v, ok := c.cache[key+"\x00"+environment]
	c.mu.RUnlock()
	if !ok {
		return fallback
	}
	return flagcontrol.Enabled(v.Tenant, v.Key, v.Environment, user, v.Enabled, v.Percentage, v.Allow)
}
