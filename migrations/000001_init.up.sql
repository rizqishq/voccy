create table if not exists boards (
    id uuid default gen_random_uuid() primary key,
    name varchar(128) not null,
    slug varchar(256) unique not null,
    description varchar(1024),
    is_active boolean default true,
    settings jsonb default '{}',
    created_at timestamptz default now(),
    updated_at timestamptz default now()
);

create table if not exists feedbacks (
    id uuid default gen_random_uuid() primary key,
    board_id uuid not null references boards(id) on delete cascade,
    title varchar(256) not null,
    body text not null,
    author_name varchar(128),
    author_email varchar(256),
    status varchar(32) default 'open',
    created_at timestamptz default now(),
    updated_at timestamptz default now()
);
