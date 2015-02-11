
DROP TABLE IF EXISTS account_data;
CREATE TABLE account_data (
  username varchar(17) NOT NULL,
  password char(64) NOT NULL,
  email varchar(255),
  registration_date date NOT NULL,
  lastip varchar(16),
  lasthwinfo tinyblob,
  guildcard int(11) NOT NULL AUTO_INCREMENT PRIMARY KEY,
  is_gm boolean  DEFAULT false,
  is_banned boolean DEFAULT false,
  is_active boolean DEFAULT false,
  team_id int(11) NOT NULL DEFAULT'-1',
  privlevel smallint(3) NOT NULL DEFAULT '0',
  lastchar tinyblob
);

-- Queried every time a user logs in.
CREATE INDEX login_index ON account_data (username, password);

CREATE TABLE player_options (
  guildcard int(11) PRIMARY KEY,
  key_config blob,
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard)
);

CREATE TABLE characters (
  guildcard int(11) PRIMARY KEY,
  slot_num tinyint(2),
  character_data blob,
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard)
);

-- This table may get big, so keep an index to make queries from paket E3 fast.
CREATE INDEX character_index ON characters(guildcard, slot_num);