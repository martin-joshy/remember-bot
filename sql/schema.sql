CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    l_id TEXT NOT NULL UNIQUE,
    phone_number TEXT UNIQUE,
    display_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    stanza_id TEXT,
    sent_at DATETIME NOT NULL,
    type TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS message_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    body TEXT,
    file_name TEXT,
    file_path TEXT,
    mime_type TEXT,
    file_size INTEGER,
    FOREIGN KEY (message_id) REFERENCES messages(id)
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, name)
);

CREATE TABLE IF NOT EXISTS message_tags (
    message_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (message_id, tag_id),
    FOREIGN KEY (message_id) REFERENCES messages(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE IF NOT EXISTS sent_tagged_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  stanza_id TEXT NOT NULL UNIQUE,
  original_message_id INTEGER NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  sent_at DATETIME NOT NULL,
  FOREIGN KEY (original_message_id) REFERENCES messages(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);



