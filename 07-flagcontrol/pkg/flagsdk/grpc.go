package flagsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func (c *Client) RefreshGRPC(ctx context.Context, conn grpc.ClientConnInterface, key, environment string) error {
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.Token)
	in, _ := structpb.NewStruct(map[string]any{"key": key, "environment": environment})
	out := new(structpb.Struct)
	if e := conn.Invoke(ctx, "/portfolio.flags.Config/Get", in, out); e != nil {
		return e
	}
	data, e := json.Marshal(out.AsMap())
	if e != nil {
		return e
	}
	var config Config
	if e = json.Unmarshal(data, &config); e != nil {
		return e
	}
	if config.Key != key || config.Environment != environment || config.Percentage < 0 || config.Percentage > 100 {
		return fmt.Errorf("invalid configuration")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = map[string]Config{}
	}
	c.cache[key+"\x00"+environment] = config
	return nil
}
