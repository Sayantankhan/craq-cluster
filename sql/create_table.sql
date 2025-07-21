CREATE TABLE IF NOT EXISTS public.chunk_metadata (
  folder STRING NOT NULL DEFAULT '/',
  file_name STRING NOT NULL,
  seq INT8 NOT NULL,
  state STRING NOT NULL,
  path STRING NOT NULL,
  CONSTRAINT pk_folder_file PRIMARY KEY (folder, file_name)
);