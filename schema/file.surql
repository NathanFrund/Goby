-- =============================================================================
-- File Table Schema
-- =============================================================================
-- This schema defines the structure for storing file metadata. The actual
-- file content is stored in a separate system (like a local disk or S3)
-- and is referenced by the `storage_path`.
-- =============================================================================

-- Define the 'file' table with a strict schema.
DEFINE TABLE file SCHEMAFULL
    PERMISSIONS
        FOR create WHERE $auth IS NOT NONE
        FOR select WHERE $auth.id = user_id
        FOR update WHERE $auth.id = user_id
        FOR delete WHERE $auth.id = user_id;

-- Define the fields for the 'file' table.
DEFINE FIELD user_id ON file TYPE record<user>
    ASSERT $value != NONE
    COMMENT "Reference to the user who owns this file";

DEFINE FIELD filename ON file TYPE string
    ASSERT $value != NONE
    COMMENT "Original name of the file";

DEFINE FIELD mime_type ON file TYPE string
    COMMENT "MIME type of the file";

DEFINE FIELD size ON file TYPE int
    ASSERT $value >= 0
    COMMENT "Size of the file in bytes";

DEFINE FIELD storage_path ON file TYPE string
    ASSERT $value != NONE
    COMMENT "Path to the file in the storage backend";

DEFINE FIELD created_at ON file TYPE datetime 
    VALUE $value OR time::now()
    COMMENT "When the file was created";

DEFINE FIELD updated_at ON file TYPE datetime 
    VALUE time::now()
    COMMENT "When the file was last updated";

-- Optional: For soft deletes
DEFINE FIELD deleted_at ON file TYPE option<datetime>
    COMMENT "When the file was deleted (if using soft deletes)";

-- Define indexes for efficient querying.
DEFINE INDEX file_storage_path_idx ON file COLUMNS storage_path UNIQUE;
DEFINE INDEX file_user_id_idx ON TABLE file COLUMNS user_id;
