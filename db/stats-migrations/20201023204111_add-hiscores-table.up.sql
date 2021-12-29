create table hiscores (
    id integer not null primary key autoincrement,
    name text not null,
    created_at integer not null
);

create table hiscore_values (
    id integer not null primary key autoincrement,
    hiscore_id integer not null,
    key text not null,
    value integer not null,
    
    foreign key (hiscore_id) references hiscores (id) on delete cascade
);

create index hiscore_values_value_idx on hiscore_values (value);
create index hiscore_values_hiscore_id_idx on hiscore_values (hiscore_id);

create table hiscore_data (
    id integer not null primary key autoincrement,
    hiscore_id integer not null,
    key text not null,
    value text not null,
    
    foreign key (hiscore_id) references hiscores (id) on delete cascade
);
