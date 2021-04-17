CREATE TABLE entries (
    id BIGSERIAL NOT NULL
        CONSTRAINT entries_pkey
            PRIMARY KEY,
    title text not null,
    content text,
    document_vectors tsvector
);

