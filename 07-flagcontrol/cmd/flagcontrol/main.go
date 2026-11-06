package main

import (
	"context"
	"flagcontrol/internal/platform"
	"flagcontrol/migrations"
	"flagcontrol/projects/flagcontrol"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	platform.Serve("flagcontrol", flagcontrol.Register, func(ctx context.Context, pool *pgxpool.Pool) error { return migrations.Apply(ctx, pool, "flagcontrol") })
}
