
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

CREATE INDEX LoginIndex ON account_data (username, password);
