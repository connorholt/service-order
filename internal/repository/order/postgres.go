package order

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolaev/service-order/internal/domain/entity"
)

// Postgres implements Repository using pgxpool and Postgres
// Table schema (created on first NewPostgres call):
//   create table if not exists orders (
//     id text primary key,
//     user_id text not null,
//     order_number text,
//     fio text,
//     restaurant_id text not null,
//     items jsonb not null,
//     total_price bigint not null,
//     address jsonb not null,
//     status text not null,
//     created_at timestamptz not null,
//     updated_at timestamptz not null,
//     estimated_delivery timestamptz not null,
//     is_deleted boolean not null default false
//   );
//   create index if not exists orders_user_id_idx on orders(user_id);

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, pool *pgxpool.Pool) (*Postgres, error) {
	p := &Postgres{pool: pool}
	if err := p.initSchema(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Postgres) initSchema(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		create table if not exists orders (
		  id text primary key,
		  user_id text not null,
		  order_number text,
		  fio text,
		  restaurant_id text not null,
		  items jsonb not null,
		  total_price bigint not null,
		  address jsonb not null,
		  status text not null,
		  created_at timestamptz not null,
		  updated_at timestamptz not null,
		  estimated_delivery timestamptz not null,
		  is_deleted boolean not null default false
		);
		create index if not exists orders_user_id_idx on orders(user_id);
	`)
	return err
}

func (p *Postgres) Create(ctx context.Context, o *entity.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	addr, err := json.Marshal(o.Address)
	if err != nil {
		return err
	}
	ct, ut, et := o.CreatedAt, o.UpdatedAt, o.EstimatedDelivery
	if ct.IsZero() {
		ct = time.Now().UTC()
	}
	if ut.IsZero() {
		ut = ct
	}
	if et.IsZero() {
		et = ct
	}
	_, err = p.pool.Exec(ctx, `
		insert into orders(
		  id, user_id, order_number, fio, restaurant_id, items, total_price, address, status, created_at, updated_at, estimated_delivery, is_deleted
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, o.ID, o.UserID, o.OrderNumber, o.FIO, o.RestaurantID, items, o.TotalPrice, addr, string(o.Status), ct, ut, et, o.IsDeleted)
	return err
}

func (p *Postgres) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	row := p.pool.QueryRow(ctx, `
		select id, user_id, order_number, fio, restaurant_id, items, total_price, address, status, created_at, updated_at, estimated_delivery, is_deleted
		from orders where id = $1
	`, id)
	var (
		o          entity.Order
		itemsBytes []byte
		addrBytes  []byte
		status     string
	)
	if err := row.Scan(&o.ID, &o.UserID, &o.OrderNumber, &o.FIO, &o.RestaurantID, &itemsBytes, &o.TotalPrice, &addrBytes, &status, &o.CreatedAt, &o.UpdatedAt, &o.EstimatedDelivery, &o.IsDeleted); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		// pgx returns ErrNoRows; map на доменную ошибку
		if err.Error() == "no rows in result set" {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(itemsBytes, &o.Items); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(addrBytes, &o.Address); err != nil {
		return nil, err
	}
	o.Status = entity.OrderStatus(status)
	return &o, nil
}

func (p *Postgres) Update(ctx context.Context, o *entity.Order) error {
	items, err := json.Marshal(o.Items)
	if err != nil {
		return err
	}
	addr, err := json.Marshal(o.Address)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		update orders set 
		  user_id=$2, order_number=$3, fio=$4, restaurant_id=$5, items=$6, total_price=$7, address=$8, status=$9, created_at=$10, updated_at=$11, estimated_delivery=$12, is_deleted=$13
		where id=$1
	`, o.ID, o.UserID, o.OrderNumber, o.FIO, o.RestaurantID, items, o.TotalPrice, addr, string(o.Status), o.CreatedAt, o.UpdatedAt, o.EstimatedDelivery, o.IsDeleted)
	return err
}

func (p *Postgres) MarkDeleted(ctx context.Context, id string, userID string) error {
	ct := time.Now().UTC()
	cmd, err := p.pool.Exec(ctx, `
		update orders set is_deleted=true, status=$3, updated_at=$4
		where id=$1 and user_id=$2
 `, id, userID, string(entity.OrderStatusDeleted), ct)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		// Distinguish not found vs foreign ownership to match in-memory behavior
		var existingUserID string
		row := p.pool.QueryRow(ctx, `select user_id from orders where id=$1`, id)
		if scanErr := row.Scan(&existingUserID); scanErr != nil {
			return entity.ErrNotFound
		}
		return entity.ErrForeignOwnership
	}
	return nil
}
