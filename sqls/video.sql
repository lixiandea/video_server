CREATE TABLE IF NOT EXISTS `video_info`(
   `id` VARCHAR(64) NOT NULL PRIMARY KEY,
   `author_id`  INT UNSIGNED,
   `name` TEXT,
   `display_ctime` TEXT,
   `create_time` DATETIME
)ENGINE=InnoDB DEFAULT CHARSET=utf8;