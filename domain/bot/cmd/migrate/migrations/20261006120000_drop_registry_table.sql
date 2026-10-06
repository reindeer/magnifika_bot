-- +goose Up
drop table registry;

-- +goose Down
create table registry (
    code varchar(50) not null unique,
    value text null
);
