create table hiscores (
    id integer not null primary key autoincrement,
    name text,
    team text,
    kills integer,
    deaths integer,
    bounty integer,
    timestamp integer
);
