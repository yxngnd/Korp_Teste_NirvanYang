CREATE TABLE IF NOT EXISTS produtos (
    id          SERIAL PRIMARY KEY,
    codigo      VARCHAR(50) UNIQUE NOT NULL,
    descricao   VARCHAR(255) NOT NULL,
    saldo       INTEGER NOT NULL CHECK (saldo >= 0),
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id          SERIAL PRIMARY KEY,
    chave       VARCHAR(100) UNIQUE NOT NULL,
    resposta    JSONB NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now()
);
