CREATE TABLE IF NOT EXISTS public.chunk_metadata (
  chunk_id UUID NOT NULL,
  file_name STRING NOT NULL,
  seq INT8 NOT NULL,
  state STRING NOT NULL,
  path STRING NOT NULL,
  CONSTRAINT chunk_metadata_pkey PRIMARY KEY (chunk_id ASC)
);