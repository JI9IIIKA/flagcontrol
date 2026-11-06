package flagcontrol

import (
	"crypto/sha256"
	"encoding/binary"
	p "flagcontrol/internal/platform"
)

func Enabled(tenant, key, environment, user string, enabled bool, percentage int, allow []string) bool {
	if !enabled {
		return false
	}
	for _, id := range allow {
		if id == user {
			return true
		}
	}
	sum := sha256.Sum256([]byte(tenant + "\x00" + environment + "\x00" + key + "\x00" + user))
	bucket := binary.BigEndian.Uint64(sum[:8]) % 10000
	return bucket < uint64(percentage*100)
}
func Register(a *p.App) {
	RegisterGRPC(a)
	a.Handle("PUT", "/flags/{key}/{environment}", func(r *p.Request) (any, error) {
		if r.Role != "admin" {
			return nil, p.Fail(403, "admin required")
		}
		var b struct {
			Version    int      `json:"expected_version"`
			Enabled    bool     `json:"enabled"`
			Percentage int      `json:"percentage"`
			Allow      []string `json:"allow_users"`
		}
		if e := r.Decode(&b); e != nil {
			return nil, e
		}
		if b.Percentage < 0 || b.Percentage > 100 {
			return nil, p.Bad("percentage must be 0..100")
		}
		if b.Allow == nil {
			b.Allow = []string{}
		}
		var current int
		if e := r.Tx.QueryRow(r.Ctx, `SELECT COALESCE((SELECT version FROM flag_heads WHERE tenant=$1 AND key=$2 AND environment=$3),0)`, r.Tenant, r.Param("key"), r.Param("environment")).Scan(&current); e != nil {
			return nil, e
		}
		if current != b.Version {
			return nil, p.Conflict("stale version")
		}
		result, e := r.One(`INSERT INTO flags(tenant,key,environment,version,enabled,percentage,allow_users) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING *`, r.Tenant, r.Param("key"), r.Param("environment"), current+1, b.Enabled, b.Percentage, b.Allow)
		if e != nil {
			return nil, e
		}
		if e = r.Exec(`INSERT INTO flag_heads(tenant,key,environment,version) VALUES($1,$2,$3,$4) ON CONFLICT(tenant,key,environment) DO UPDATE SET version=excluded.version`, r.Tenant, r.Param("key"), r.Param("environment"), current+1); e != nil {
			return nil, e
		}
		return result, r.Event("flag.updated", result)
	})
	a.Handle("GET", "/flags/{key}/{environment}", func(r *p.Request) (any, error) {
		return r.One(`SELECT f.* FROM flags f JOIN flag_heads h USING(tenant,key,environment,version) WHERE f.tenant=$1 AND f.key=$2 AND f.environment=$3`, r.Tenant, r.Param("key"), r.Param("environment"))
	})
	a.Handle("GET", "/flags/{key}/{environment}/history", func(r *p.Request) (any, error) {
		return r.All(`SELECT * FROM flags WHERE tenant=$1 AND key=$2 AND environment=$3 ORDER BY version DESC LIMIT 100`, r.Tenant, r.Param("key"), r.Param("environment"))
	})
	a.Handle("POST", "/flags/{key}/{environment}/rollback", func(r *p.Request) (any, error) {
		if r.Role != "admin" {
			return nil, p.Fail(403, "admin required")
		}
		var b struct {
			Version  int `json:"version"`
			Expected int `json:"expected_version"`
		}
		if e := r.Decode(&b); e != nil {
			return nil, e
		}
		var current int
		if e := r.Tx.QueryRow(r.Ctx, `SELECT version FROM flag_heads WHERE tenant=$1 AND key=$2 AND environment=$3`, r.Tenant, r.Param("key"), r.Param("environment")).Scan(&current); e != nil {
			return nil, e
		}
		if current != b.Expected {
			return nil, p.Conflict("stale version")
		}
		result, e := r.One(`INSERT INTO flags(tenant,key,environment,version,enabled,percentage,allow_users) SELECT tenant,key,environment,$4,enabled,percentage,allow_users FROM flags WHERE tenant=$1 AND key=$2 AND environment=$3 AND version=$5 RETURNING *`, r.Tenant, r.Param("key"), r.Param("environment"), current+1, b.Version)
		if e != nil {
			return nil, e
		}
		if e = r.Exec(`UPDATE flag_heads SET version=$4 WHERE tenant=$1 AND key=$2 AND environment=$3`, r.Tenant, r.Param("key"), r.Param("environment"), current+1); e != nil {
			return nil, e
		}
		return result, r.Event("flag.rolled_back", result)
	})
	a.Handle("GET", "/evaluate/{key}/{environment}", func(r *p.Request) (any, error) {
		user := r.HTTP.URL.Query().Get("user")
		if user == "" {
			return nil, p.Bad("user required")
		}
		var enabled bool
		var percentage, version int
		var allow []string
		if e := r.Tx.QueryRow(r.Ctx, `SELECT f.enabled,f.percentage,f.version,f.allow_users FROM flags f JOIN flag_heads h USING(tenant,key,environment,version) WHERE f.tenant=$1 AND f.key=$2 AND f.environment=$3`, r.Tenant, r.Param("key"), r.Param("environment")).Scan(&enabled, &percentage, &version, &allow); e != nil {
			return nil, e
		}
		return map[string]any{"enabled": Enabled(r.Tenant, r.Param("key"), r.Param("environment"), user, enabled, percentage, allow), "version": version}, nil
	})
}
