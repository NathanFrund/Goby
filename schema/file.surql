-- =============================================================================
-- File Table Schema
-- =============================================================================
-- This schema defines the structure for storing file metadata. The actual
-- file content is stored in a separate system (like a local disk or S3)
-- and is referenced by the `storage_path`.
-- =============================================================================

-- Define the 'file' table with a strict schema.
-- This ensures that only the fields we define can be created.
DEFINE TABLE file SCHEMAFULL;

-- Define the fields for the 'file' table.
DEFINE FIELD user_id ON file TYPE record(user)
    -- Ensure that a user is always associated with a file.
    ASSERT $value != NONE;

DEFINE FIELD filename ON file TYPE string
    ASSERT $value != NONE;

DEFINE FIELD mime_type ON file TYPE string;

DEFINE FIELD size_bytes ON file TYPE int
    ASSERT $value >= 0;

DEFINE FIELD storage_path ON file TYPE string
    -- The storage path must be unique across all files.
    ASSERT $value != NONE;

DEFINE FIELD created_at ON file TYPE datetime VALUE $value OR time::now();
DEFINE FIELD updated_at ON file TYPE datetime VALUE time::now();

-- Optional: For soft deletes
DEFINE FIELD deleted_at ON file TYPE option<datetime>;

-- Define indexes for efficient querying.
DEFINE INDEX file_storage_path_idx ON file COLUMNS storage_path UNIQUE;
DEFINE INDEX file_user_id_idx ON file COLUMNS user_id;