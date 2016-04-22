
DROP TABLE IF EXISTS account_data;
CREATE TABLE account_data (
  username varchar(17) NOT NULL,
  password char(64) NOT NULL,
  email varchar(255),
  registration_date timestamp DEFAULT NOW(),
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
  guildcard int(11),
  slot_num tinyint(2),
  experience int DEFAULT 0,
  level smallint DEFAULT 0,
  guildcard_str binary(16),
  name_color int unsigned DEFAULT X'FFFFFFFF',
  model smallint,
  name_color_chksm int,
  section_id tinyint,
  char_class tinyint,
  v2_flags tinyint,
  version tinyint,
  v1_flags int,
  costume smallint,
  skin smallint,
  face smallint,
  head smallint,
  hair smallint,
  hair_red smallint,
  hair_green smallint,
  hair_blue smallint,
  proportion_x float,
  proportion_y float,
  name binary(24),
  playtime int DEFAULT 0,
  # keyConfig binary(232),
  # techniques blob,
  # options blob,
  atp smallint,
  mst smallint,
  evp smallint,
  hp smallint,
  dfp smallint,
  ata smallint,
  lck smallint,
  meseta int,
  bank_use int DEFAULT 0,
  bank_meseta int DEFAULT 0,
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard)
);

-- Keep an index to make queries from paket E3 fast.
CREATE INDEX character_index ON characters(guildcard, slot_num);

CREATE TABLE guildcard_entries (
  guildcard int(11) PRIMARY KEY,
  friend_gc int(11) NOT NULL,
  name binary(48),
  team_name binary(32),
  description binary(176),
  language tinyint,
  section_id tinyint,
  char_class tinyint,
  comment binary(176),
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard),
  FOREIGN KEY (friend_gc) REFERENCES account_data(guildcard)
);