
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
  atp smallint,
  mst smallint,
  evp smallint,
  hp smallint,
  dfp smallint,
  tp smallint,
  lck smallint,
  ata smallint,
  level smallint,
  experience int,
  meseta int,
  name_color_blue tinyint,
  name_color_green tinyint,
  name_color_red tinyint,
  name_color_opacity tinyint,
  skin_id smallint,
  section_id tinyint,
  char_class tinyint,
  skin_flag tinyint,
  costume smallint,
  skin smallint,
  face smallint,
  head smallint,
  hair smallint,
  hair_color_red smallint,
  hair_color_blue smallint,
  hair_color_green smallint,
  proportion_x int,
  proportion_y int,
  name blob,
  playtime int,
  unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
  techniques blob,
  options blob,
  bank_use int,
  bank_meseta int,
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard)
);

-- This table may get big, so keep an index to make queries from paket E3 fast.
CREATE INDEX character_index ON characters(guildcard, slot_num);

CREATE TABLE guildcard_entries (
  guildcard int(11) PRIMARY KEY,
  friend_gc int(11) NOT NULL,
  name blob,
  team_name blob,
  description blob,
  language tinyint,
  section_id tinyint,
  char_class tinyint,
  comment blob,
  FOREIGN KEY (guildcard) REFERENCES account_data(guildcard),
  FOREIGN KEY (friend_gc) REFERENCES account_data(guildcard)
);