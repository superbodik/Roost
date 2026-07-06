CREATE TABLE database_hosts (
    id                      BIGSERIAL PRIMARY KEY,
    name                    TEXT NOT NULL,
    host                    TEXT NOT NULL,
    port                    INT NOT NULL DEFAULT 3306,
    admin_username          TEXT NOT NULL,
    admin_password_encrypted TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE server_databases
    ADD CONSTRAINT server_databases_host_fk
    FOREIGN KEY (database_host_id) REFERENCES database_hosts(id);
