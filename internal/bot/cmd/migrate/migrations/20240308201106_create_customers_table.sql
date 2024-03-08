-- +goose Up
create table customers (
    customer_id int not null unique,
    phone varchar(255) not null
);

-- +goose Down
drop table customers
