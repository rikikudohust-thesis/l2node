```sql
create table if not exists accounts (
	item_id serial,
	idx bigint,
	token_id int not null,
	batch_num bigint not null,
	bjj bytea not null,
	eth_addr bytea not null,
	primary key(idx)
);
create table if not exists account_updates (
	item_id SERIAL,
	eth_block_num bigint not null,
	batch_num bigint not null,
	idx bigint not null,
	nonce bigint not null,
	balance numeric not null
);

create table if not exists blocks (
	eth_block_num bigint,
	timestamp int not null,
	hash bytea not null,
	primary key(eth_block_num)
);

create table if not exists batches (
	item_id serial ,
  batch_num bigint unique not null,
  eth_block_num bigint not null,
  forger_addr bytea, -- fake foreign key for coordinator
  fees_collected bytea,
  fee_idxs_coordinator bytea,
  state_root numeric not null,
  num_accounts bigint not NULL,
	last_idx bigint not null,
  exit_root numeric not null,
  forge_l1_txs_num bigint,
  slot_num bigint,
  total_fees_usd numeric default 0,
	gas_price numeric default 0,
	gas_usd numeric default 0,
	ether_price_usd numeric default 0,
	primary key(item_id)
);

create table if not exists tokens (
	item_id serial,
	token_id int unique not null,
	eth_block_num bigint,
	eth_addr bytea not null,
	name text,
	symbol text,
	decimals int,
	usd numeric,
	usd_update int,
	primary key(item_id)
);

create table if not exists exit_trees (
  item_id serial primary key,
  batch_num bigint,
  account_idx bigint,
  merkle_proof bytea ,
  balance decimal(78,0),
  instant_withdrawn bigint,
  delayed_withdraw_request bigint,
  owner bytea,
  token bytea,
  delayed_withdrawn bigint
);

create table if not exists txes(
  item_id bigserial primary key,
  is_l1 boolean not null,
  id bytea,
  type varchar(40) not null,
  position int not null,
  from_idx bigint,
  effective_from_idx bigint,
  from_eth_addr bytea,
  from_bjj bytea,
  to_idx bigint not null,
  to_eth_addr bytea,
  to_bjj bytea,
  amount numeric not null,
  amount_success boolean not null default true,
  amount_f numeric not null,
  token_id int not null,
  amount_usd numeric, -- value of the amount in usd at the moment the tx was inserted in the db
  batch_num bigint, -- can be null in the case of l1 txs that are on the queue but not forged yet.
  eth_block_num bigint not null,
  eth_tx_hash bytea,
    -- l1
  to_forge_l1_txs_num bigint,
  user_origin boolean,
  deposit_amount numeric,
  deposit_amount_success boolean not null default true,
  deposit_amount_f numeric,
  deposit_amount_usd numeric,
  l1_fee numeric,
    -- l2
  fee int,
  fee_usd numeric,
  nonce bigint
);

create table if not exists tx_pool (
  item_id serial,
  tx_id bytea unique,
  from_idx bigint not null,
  effective_from_eth_addr bytea,
  effective_from_bjj bytea,
  to_idx bigint,
  to_eth_addr bytea,
  to_bjj bytea,
  effective_to_eth_addr bytea,
  effective_to_bjj bytea,
  token_id int not null,
  amount numeric not null,
  amount_f numeric not null,
  fee smallint not null,
  nonce bigint not null,
  state char(4) not null,
  info varchar,
  error_code numeric,
  error_type varchar,
  signature bytea not null,
  timestamp int,
  batch_num bigint,
  rq_from_idx bigint,
  rq_to_idx bigint,
  rq_to_eth_addr bytea,
  rq_to_bjj bytea,
  rq_token_id int,
  rq_amount numeric,
  rq_fee smallint,
  rq_nonce bigint,
  rq_offset smallint default null,
  atomic_group_id bytea default null,
  tx_type varchar(40) not null,
  max_num_batch bigint default null,
  client_ip varchar,
  external_delete boolean not null default false,
  primary key(item_id)
);

create table if not exists txes_l2(
  item_id bigserial primary key,
  is_l1 boolean not null,
  id bytea,
  type varchar(40) not null,
  position int not null,
  from_idx bigint,
  effective_from_idx bigint,
  from_eth_addr bytea,
  from_bjj bytea,
  to_idx bigint not null,
  to_eth_addr bytea,
  to_bjj bytea,
  amount numeric not null,
  amount_success boolean not null default true,
  amount_f numeric not null,
  token_id int not null,
  amount_usd numeric, -- value of the amount in usd at the moment the tx was inserted in the db
  batch_num bigint, -- can be null in the case of l1 txs that are on the queue but not forged yet.
  eth_block_num bigint not null,
  eth_tx_hash bytea,
    -- l1
  to_forge_l1_txs_num bigint,
  user_origin boolean,
  deposit_amount numeric,
  deposit_amount_success boolean not null default true,
  deposit_amount_f numeric,
  deposit_amount_usd numeric,
  l1_fee numeric,
    -- l2
  fee int,
  fee_usd numeric,
  nonce bigint
);

create table if not exists block_l2(
	eth_block_num bigint,
	is_l1 boolean,
	batch_num bigint,
	primary key(batch_num)
);


create table if not exists accounts_l2 (
	item_id serial,
	idx bigint,
	token_id int not null,
	batch_num bigint not null,
	bjj bytea not null,
	eth_addr bytea not null,
	primary key(idx)
);

create table if not exists account_updates_l2 (
	item_id SERIAL,
	eth_block_num bigint not null,
	batch_num bigint not null,
	idx bigint not null,
	nonce bigint not null,
	balance numeric not null
);

create table if not exists txes_queue(
  item_id bigserial primary key,
  is_l1 boolean not null,
  id bytea,
  type varchar(40) not null,
  position int not null,
  from_idx bigint,
  effective_from_idx bigint,
  from_eth_addr bytea,
  from_bjj bytea,
  to_idx bigint not null,
  to_eth_addr bytea,
  to_bjj bytea,
  amount numeric not null,
  amount_success boolean not null default true,
  amount_f numeric not null,
  token_id int not null,
  amount_usd numeric, -- value of the amount in usd at the moment the tx was inserted in the db
  batch_num bigint, -- can be null in the case of l1 txs that are on the queue but not forged yet.
  eth_block_num bigint not null,
  eth_tx_hash bytea,
    -- l1
  to_forge_l1_txs_num bigint,
  user_origin boolean,
  deposit_amount numeric,
  deposit_amount_success boolean not null default true,
  deposit_amount_f numeric,
  deposit_amount_usd numeric,
  l1_fee numeric,
    -- l2
  fee int,
  fee_usd numeric,
  nonce bigint
);

alter table txes add constraint tx_id_unique unique (id);
alter table txes_l2 add constraint tx_id_unique_l2 unique (id);
alter table txes_queue add constraint tx_id_unique_txes_order unique (id);
alter table exit_trees add constraint exit_trees_unique unique (account_idx, batch_num);

```
## Trunscade data
```sql
TRUNCATE account_updates RESTART IDENTITY CASCADE;
TRUNCATE accounts RESTART IDENTITY CASCADE;
TRUNCATE account_updates_l2 RESTART IDENTITY CASCADE;
TRUNCATE accounts_l2 RESTART IDENTITY CASCADE;
TRUNCATE batches RESTART IDENTITY CASCADE;
TRUNCATE block_l2 RESTART IDENTITY CASCADE;
TRUNCATE blocks RESTART IDENTITY CASCADE;
TRUNCATE exit_trees RESTART IDENTITY CASCADE;
TRUNCATE tokens RESTART IDENTITY CASCADE;
TRUNCATE tx_pool RESTART IDENTITY CASCADE;
TRUNCATE txes RESTART IDENTITY CASCADE;
TRUNCATE txes_l2 RESTART IDENTITY CASCADE;
```

## Init genesis value
```sql
insert into tokens (
    token_id,
    eth_block_num,
    eth_addr,
    name,
    symbol,
    decimals
) values (
    0,
    0,
    '\x0000000000000000000000000000000000000000',
    'ether',
    'eth',
    18
);

```
