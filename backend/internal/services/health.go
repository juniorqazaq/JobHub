package services

import (
	"context"
	"time"
)

type DatabasePinger interface{ Ping(context.Context) error }
type HealthService struct{ database DatabasePinger }

func NewHealthService(database DatabasePinger) *HealthService {
	return &HealthService{database: database}
}
func (s *HealthService) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.database.Ping(ctx)
}
