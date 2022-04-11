create table orders
(
    id        varchar not null
        constraint orders_pk
            primary key,
    data_json text
);

alter table orders
    owner to postgres;

create unique index orders_id_uindex
    on orders (id);