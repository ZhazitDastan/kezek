-- +goose Up

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- 1. Бизнесы
CREATE TABLE businesses (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug          TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    address       TEXT,
    timezone      TEXT        NOT NULL DEFAULT 'Asia/Almaty',  -- валидируется в Go (time.LoadLocation)
    hold_duration INTERVAL    NOT NULL DEFAULT '10 minutes',
    cancel_before INTERVAL    NOT NULL DEFAULT '2 hours',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT businesses_slug_format   CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT businesses_hold_positive CHECK (hold_duration > INTERVAL '0'),
    CONSTRAINT businesses_cancel_nonneg CHECK (cancel_before >= INTERVAL '0')
);

-- 2. Услуги
CREATE TABLE services (
    id          BIGINT   GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT   NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    name        TEXT     NOT NULL,
    duration    INTERVAL NOT NULL,
    price       BIGINT   NOT NULL,  -- в тиынах
    is_active   BOOLEAN  NOT NULL DEFAULT true,

    CONSTRAINT services_duration_positive CHECK (duration > INTERVAL '0'),
    CONSTRAINT services_price_nonneg      CHECK (price >= 0),
    CONSTRAINT services_id_business_uq    UNIQUE (id, business_id)
);

CREATE INDEX services_business_id_idx ON services (business_id);

-- 3. Мастера
CREATE TABLE staff (
    id          BIGINT  GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT  NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT staff_id_business_uq UNIQUE (id, business_id)
);

CREATE INDEX staff_business_id_idx ON staff (business_id);

-- 4. Какие услуги делает мастер (мастер и услуга — из одного бизнеса)
CREATE TABLE staff_services (
    business_id BIGINT NOT NULL,
    staff_id    BIGINT NOT NULL,
    service_id  BIGINT NOT NULL,

    CONSTRAINT staff_services_pk         PRIMARY KEY (staff_id, service_id),
    CONSTRAINT staff_services_staff_fk   FOREIGN KEY (staff_id, business_id)
        REFERENCES staff (id, business_id) ON DELETE CASCADE,
    CONSTRAINT staff_services_service_fk FOREIGN KEY (service_id, business_id)
        REFERENCES services (id, business_id) ON DELETE CASCADE
);

CREATE INDEX staff_services_service_id_idx ON staff_services (service_id);

-- 5. Клиенты
CREATE TABLE clients (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT      NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    phone       TEXT        NOT NULL,
    telegram_id BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT clients_business_phone_uq    UNIQUE (business_id, phone),
    CONSTRAINT clients_business_telegram_uq UNIQUE (business_id, telegram_id),
    CONSTRAINT clients_id_business_uq       UNIQUE (id, business_id)
);

-- 6. Бронирования
CREATE TABLE bookings (
    id              BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id     BIGINT      NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    staff_id        BIGINT      NOT NULL,
    service_id      BIGINT      NOT NULL,
    client_id       BIGINT      NOT NULL,
    period          TSTZRANGE   NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'held',
    price           BIGINT      NOT NULL,  -- в тиынах, копия цены услуги
    hold_expires_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- мастер, услуга и клиент должны быть из того же бизнеса, что и бронь
    CONSTRAINT bookings_staff_fk   FOREIGN KEY (staff_id, business_id)
        REFERENCES staff (id, business_id),
    CONSTRAINT bookings_service_fk FOREIGN KEY (service_id, business_id)
        REFERENCES services (id, business_id),
    CONSTRAINT bookings_client_fk  FOREIGN KEY (client_id, business_id)
        REFERENCES clients (id, business_id),

    CONSTRAINT bookings_status_check CHECK (
        status IN ('held', 'confirmed', 'completed', 'cancelled', 'no_show', 'expired')
    ),
    CONSTRAINT bookings_price_nonneg CHECK (price >= 0),

    -- интервал непустой, конечный и в формате [начало, конец)
    CONSTRAINT bookings_period_valid CHECK (
        NOT isempty(period)
        AND NOT lower_inf(period) AND NOT upper_inf(period)
        AND lower_inc(period) AND NOT upper_inc(period)
    ),

    -- бронь в статусе held обязана иметь срок истечения
    CONSTRAINT bookings_hold_has_expiry CHECK (
        status <> 'held' OR hold_expires_at IS NOT NULL
    ),

    -- у одного мастера не может быть пересекающихся активных броней
    CONSTRAINT bookings_no_overlap EXCLUDE USING gist (
        staff_id WITH =,
        period   WITH &&
    ) WHERE (status IN ('held', 'confirmed'))
);

CREATE INDEX bookings_client_id_idx    ON bookings (client_id);
CREATE INDEX bookings_hold_expires_idx ON bookings (hold_expires_at) WHERE status = 'held';

-- +goose Down
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS staff_services;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS businesses;
