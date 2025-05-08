package storage

import (
	"LoadBalancer/internal/config"
	"LoadBalancer/internal/domain"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
)

// Clients - структура для хранения емкости и скорости пополнения

type PostgresRepo struct {
	conn *pgxpool.Pool
}

func NewPostgresRepo(ctx context.Context, cfg config.Config) *PostgresRepo {
	DSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Username, cfg.Password, cfg.Database)
	conn, err := pgxpool.New(ctx, DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err = conn.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	return &PostgresRepo{
		conn: conn,
	}
}

func (p *PostgresRepo) Close() {
	p.conn.Close()
}

// Получениение кастомных конфигураций клиентов
func (p *PostgresRepo) GetConfig(ctx context.Context) ([]domain.Client, error) {
	query := `SELECT ip, capacity, refill_rate FROM config`
	clients := make([]domain.Client, 0)
	c := domain.Client{}
	rows, _ := p.conn.Query(ctx, query)
	_, err := pgx.ForEachRow(rows, []any{&c.ID, &c.Capacity, &c.RefillRate}, func() error {
		clients = append(clients, c)
		return nil
	})
	if err != nil {
		return clients, fmt.Errorf("failed to get config : %w", err)
	}
	return clients, nil
}
