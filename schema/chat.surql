-- Messages are the core of the chat system.
DEFINE TABLE message SCHEMAFULL;
DEFINE FIELD author ON message TYPE record<user>;
DEFINE FIELD content ON message TYPE string;
DEFINE FIELD timestamp ON message TYPE datetime VALUE $before OR time::now();
DEFINE FIELD conversation ON message TYPE record<conversation>;
DEFINE FIELD parent ON message TYPE option<record<message>>;
DEFINE FIELD edited_at ON message TYPE option<datetime>;
DEFINE FIELD deleted_at ON message TYPE option<datetime>;

-- Reactions allow users to add emojis to messages.
DEFINE TABLE reaction SCHEMAFULL;
DEFINE FIELD message ON reaction TYPE record<message>;
DEFINE FIELD user ON reaction TYPE record<user>;
DEFINE FIELD emoji ON reaction TYPE string;

-- A conversation can be a DM or a named group channel
DEFINE TABLE conversation SCHEMAFULL;
DEFINE FIELD name ON conversation TYPE option<string>; -- e.g., #general, or null for DMs
DEFINE FIELD type ON conversation TYPE string ASSERT $value IN ['dm', 'group'];
DEFINE FIELD created_at ON conversation TYPE datetime VALUE time::now();

-- Use a graph edge to model membership. This is very powerful.
-- RELATE user:id -> is_member -> conversation:id;
DEFINE TABLE is_member SCHEMAFULL;
DEFINE FIELD joined_at ON is_member TYPE datetime VALUE time::now();
