-- SEQUENCE dedicada para a numeração da nota fiscal. Usar uma sequence do
-- Postgres em vez de calcular "MAX(numero) + 1" na aplicação evita condição
-- de corrida entre duas criações de nota simultâneas.
CREATE SEQUENCE IF NOT EXISTS notas_fiscais_numero_seq START WITH 1;

CREATE TABLE IF NOT EXISTS notas_fiscais (
    id          SERIAL PRIMARY KEY,
    numero      INTEGER UNIQUE NOT NULL DEFAULT nextval('notas_fiscais_numero_seq'),
    status      VARCHAR(20) NOT NULL DEFAULT 'Aberta' CHECK (status IN ('Aberta', 'Fechada')),
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    fechada_em  TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS nota_itens (
    id              SERIAL PRIMARY KEY,
    nota_id         INTEGER NOT NULL REFERENCES notas_fiscais(id),
    produto_codigo  VARCHAR(50) NOT NULL,
    quantidade      INTEGER NOT NULL CHECK (quantidade > 0)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id          SERIAL PRIMARY KEY,
    chave       VARCHAR(100) UNIQUE NOT NULL,
    resposta    JSONB NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now()
);
