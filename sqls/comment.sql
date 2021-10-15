CREATE TABLE IF NOT EXISTS `comments`(
   `id` VARCHAR(64) NOT NULL PRIMARY KEY,
   `author_id`  INT UNSIGNED,
   `video_id` VARCHAR(64),
   `content` TEXT,
   `time` DATETIME
)ENGINE=InnoDB DEFAULT CHARSET=utf8;