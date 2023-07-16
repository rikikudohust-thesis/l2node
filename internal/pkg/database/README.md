
### Initialize database
```sql
CREATE TABLE txes (
    -- Generic TX
    item_id INTEGER PRIMARY KEY DEFAULT nextval('tx_item_id'),
    is_l1 BOOLEAN NOT NULL,
    id BYTEA,
    type VARCHAR(40) NOT NULL,
    position INT NOT NULL,
    from_idx BIGINT,
    effective_from_idx BIGINT REFERENCES account (idx) ON DELETE SET NULL,
    from_eth_addr BYTEA,
    from_bjj BYTEA,
    to_idx BIGINT NOT NULL,
    to_eth_addr BYTEA,
    to_bjj BYTEA,
    amount DECIMAL(78,0) NOT NULL,
    amount_success BOOLEAN NOT NULL DEFAULT true,
    amount_f NUMERIC NOT NULL,
    token_id INT NOT NULL REFERENCES token (token_id),
    amount_usd NUMERIC, -- Value of the amount in USD at the moment the tx was inserted in the DB
    batch_num BIGINT REFERENCES batch (batch_num) ON DELETE SET NULL, -- Can be NULL in the case of L1 txs that are on the queue but not forged yet.
    eth_block_num BIGINT NOT NULL REFERENCES block (eth_block_num) ON DELETE CASCADE,
    -- L1
    to_forge_l1_txs_num BIGINT,
    user_origin BOOLEAN,
    deposit_amount DECIMAL(78,0),
    deposit_amount_success BOOLEAN NOT NULL DEFAULT true,
    deposit_amount_f NUMERIC,
    deposit_amount_usd NUMERIC,
    -- L2
    fee INT,
    fee_usd NUMERIC,
    nonce BIGINT
);

CREATE TABLE tokens (
    item_id SERIAL PRIMARY KEY,
    token_id INT UNIQUE NOT NULL,
    eth_block_num BIGINT NOT NULL REFERENCES block (eth_block_num) ON DELETE CASCADE,
    eth_addr BYTEA UNIQUE NOT NULL,
    name VARCHAR(20) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    decimals INT NOT NULL,
    usd NUMERIC, -- value of a normalized token (1 token = 10^decimals units)
    usd_update TIMESTAMP WITHOUT TIME ZONE
);

create table if not exists accounts (
 idx bigint,
 token_id integer,
 batch_num bigint,
 bjj bytea,
 eth_addr bytea,
 nonce bigint,
 balance text,
primary key(idx)
);

create table if not exists blocks (
num bigint,
"timestamp" integer,
hash bytea,
primary key(num)
);

create table if not exists tokens (
 token_id int,
 eth_block_num bigint,
 eth_addr bytea,
 name text,
 symbol text,
 decimals int,
 primary key(token_id)
);

create table if not exists batches (
 batch_num bigint,
 eth_tx_hash bytea,
 eth_block_num bigint,
 forger_addr bytea,
 fees_collected bytea,
 fee_idxs_coordinator bytea,
 state_root text,
 num_accounts int,
 last_idx int,
 exit_root text,
 gas_used int,
 gas_price text,
 eth_price_usd float,
 forge_l1_txs_num int,
 total_fee_usd float,
 primary key(batch_num)
);

create table if not exists tx_pool (
 tx_id bytea,
 from_idx bigint,
 to_idx bigint,
 aux_to_idx bigint,
 to_eth_addr bytea,
 to_bjj bytea,
 effective_to_eth_addr bytea,
 effective_to_bjj bytea,
 token_id int,
 amount text,
 amount_f text,
 fee int,
 nonce bigint,
 state text,
 max_num_batch int,
 info text,
 signature bytea,
 timestamp int,
 rq_from_idx bigint,
 rq_to_idx bigint,
 rq_to_eth_addr bytea,
 rq_to_bjj bytea,
 rq_token_id int,
 rq_amount text,
 rq_fee int
 rq_nonce bigint,
 rq_offset smallint,
 atomic_group_id smallint,
 type text,
 client_ip text,
 primary key(tx_id)
)
```
```sql
TRUNCATE account_updates RESTART IDENTITY CASCADE;
TRUNCATE account_updates_l2 RESTART IDENTITY CASCADE;
TRUNCATE accounts RESTART IDENTITY CASCADE;
TRUNCATE accounts_l2 RESTART IDENTITY CASCADE;
TRUNCATE block_l2 RESTART IDENTITY CASCADE;
TRUNCATE exit_trees RESTART IDENTITY CASCADE;
TRUNCATE batches RESTART IDENTITY CASCADE;
TRUNCATE blocks RESTART IDENTITY CASCADE;
TRUNCATE tokens RESTART IDENTITY CASCADE;
TRUNCATE tx_pool RESTART IDENTITY CASCADE;
TRUNCATE txes RESTART IDENTITY CASCADE;
TRUNCATE txes_l2 RESTART IDENTITY CASCADE;
```
Database for l2
```sql
create table if not exists txes_l2 (
    item_id serial INTEGER PRIMARY KEY ,
    is_l1 BOOLEAN NOT NULL,
    tx_id BYTEA,
    type VARCHAR(40) NOT NULL,
    position INT NOT NULL,
    from_idx BIGINT,
    effective_from_idx BIGINT , 
    from_eth_addr BYTEA,
    from_bjj BYTEA,
    to_idx BIGINT NOT NULL,
    to_eth_addr BYTEA,
    to_bjj BYTEA,
    amount DECIMAL(78,0) NOT NULL,
    amount_success BOOLEAN NOT NULL DEFAULT true,
    amount_float NUMERIC NOT NULL,
    token_id INT NOT NULL REFERENCES token (token_id),
    amount_usd NUMERIC, -- Value of the amount in USD at the moment the tx was inserted in the DB
    batch_num BIGINT REFERENCES batch (batch_num) ON DELETE SET NULL, -- Can be NULL in the case of L1 txs that are on the queue but not forged yet.
    eth_block_num BIGINT NOT NULL REFERENCES block (eth_block_num) ON DELETE CASCADE,
    -- L1
    to_forge_l1_txs_num BIGINT,
    user_origin BOOLEAN,
    deposit_amount DECIMAL(78,0),
    deposit_amount_success BOOLEAN NOT NULL DEFAULT true,
    deposit_amount_f NUMERIC,
    deposit_amount_usd NUMERIC,
    -- L2
    fee INT,
    fee_usd NUMERIC,
    nonce BIGINT

)

```
