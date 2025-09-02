DEFINE TABLE IF NOT EXISTS user SCHEMAFULL
  PERMISSIONS FOR select, update, delete WHERE id = $auth.id;

DEFINE FIELD IF NOT EXISTS name ON TABLE user TYPE option<string>;
DEFINE FIELD IF NOT EXISTS email ON user TYPE string ASSERT string::is::email($value);
DEFINE FIELD IF NOT EXISTS password ON user TYPE string;
DEFINE INDEX IF NOT EXISTS email ON user FIELDS email UNIQUE;

-- Add the password reset token field (nullable string)
DEFINE FIELD IF NOT EXISTS resetToken ON TABLE user TYPE option<string>;

-- Add the password reset token expiry field (stored as ISO 8601 string)
DEFINE FIELD IF NOT EXISTS resetTokenExpires ON TABLE user TYPE option<string>;

-- Remove the access method if it exists, to allow re-defining it. This ensures the script is idempotent.
REMOVE ACCESS account ON DATABASE;

DEFINE ACCESS account ON DATABASE TYPE RECORD
  SIGNUP ( CREATE user SET email = $email, password = crypto::argon2::generate($password) )
  SIGNIN ( SELECT * FROM user WHERE email = $email AND crypto::argon2::compare(password, $password) )
  DURATION FOR TOKEN 15m, FOR SESSION 12h;