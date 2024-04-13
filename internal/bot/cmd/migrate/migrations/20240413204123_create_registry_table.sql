-- +goose Up
create table registry (
    code varchar(50) not null unique,
    value text null
);

-- +goose Down

drop table registry
