DROP TABLE IF EXISTS `account_data`;
CREATE TABLE `account_data` (
  `username` varchar(18) collate latin1_general_ci NOT NULL default '',
  `password` varchar(33) collate latin1_general_ci NOT NULL default '',
  `email` varchar(255) collate latin1_general_ci NOT NULL default '',
  `regtime` int(11) unsigned NOT NULL default '0',
  `lastip` varchar(16) collate latin1_general_ci NOT NULL default '',
  `lasthwinfo` tinyblob,
  `guildcard` int(11) NOT NULL auto_increment,
  `is_gm` tinyint(1) NOT NULL default '0',
  `is_banned` tinyint(1) NOT NULL default '0',
  `is_logged` tinyint(1) NOT NULL default '0',
  `is_active` tinyint(1) NOT NULL default '0',
  `team_id` int(11) NOT NULL default '-1',
  `privlevel` smallint(3) NOT NULL default '0',
  `lastchar` tinyblob,
  PRIMARY KEY  (`guildcard`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `bank_data`
-- 

DROP TABLE IF EXISTS `bank_data`;
CREATE TABLE `bank_data` (
  `guildcard` int(11) NOT NULL default '0',
  `data` blob NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `character_data`
-- 

DROP TABLE IF EXISTS `character_data`;
CREATE TABLE `character_data` (
  `guildcard` int(11) NOT NULL default '0',
  `slot` tinyint(4) NOT NULL default '0',
  `data` blob NOT NULL,
  `header` tinyblob NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `guild_data`
-- 

DROP TABLE IF EXISTS `guild_data`;
CREATE TABLE `guild_data` (
  `accountid` int(11) NOT NULL default '0',
  `friendid` int(11) NOT NULL default '0',
  `friendname` tinyblob NOT NULL,
  `friendtext` blob NOT NULL,
  `reserved` smallint(6) NOT NULL default '257',
  `sectionid` smallint(6) NOT NULL default '0',
  `class` smallint(6) NOT NULL default '0',
  `comment` tinyblob NOT NULL,
  `priority` smallint(6) NOT NULL default '0'
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `hw_bans`
-- 

DROP TABLE IF EXISTS `hw_bans`;
CREATE TABLE `hw_bans` (
  `guildcard` int(11) NOT NULL,
  `hwinfo` tinyblob NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_bin;

-- --------------------------------------------------------

-- 
-- Table structure for table `ip_bans`
-- 

DROP TABLE IF EXISTS `ip_bans`;
CREATE TABLE `ip_bans` (
  `ipinfo` varchar(16) collate latin1_general_ci NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `key_data`
-- 

DROP TABLE IF EXISTS `key_data`;
CREATE TABLE `key_data` (
  `guildcard` int(11) NOT NULL default '0',
  `controls` blob NOT NULL,
  PRIMARY KEY  (`guildcard`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `security_data`
-- 

DROP TABLE IF EXISTS `security_data`;
CREATE TABLE `security_data` (
  `guildcard` int(11) NOT NULL default '0',
  `thirtytwo` int(11) NOT NULL default '0',
  `sixtyfour` tinyblob NOT NULL,
  `slotnum` tinyint(4) NOT NULL default '-1',
  `isgm` int(11) NOT NULL default '0'
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `ship_data`
-- 

DROP TABLE IF EXISTS `ship_data`;
CREATE TABLE `ship_data` (
  `rc4key` tinyblob NOT NULL,
  `idx` int(11) NOT NULL auto_increment,
  PRIMARY KEY  (`idx`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

-- --------------------------------------------------------

-- 
-- Table structure for table `team_data`
-- 

DROP TABLE IF EXISTS `team_data`;
CREATE TABLE `team_data` (
  `name` tinyblob NOT NULL,
  `owner` int(11) NOT NULL default '0',
  `flag` blob NOT NULL,
  `teamid` int(11) NOT NULL auto_increment,
  PRIMARY KEY  (`teamid`)
) ENGINE=InnoDB  DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;
