CREATE DATABASE IF NOT EXISTS craftsail_growth CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'craftsail'@'%' IDENTIFIED BY 'CHANGE-ME';
CREATE USER IF NOT EXISTS 'craftsail'@'localhost' IDENTIFIED BY 'CHANGE-ME';
GRANT ALL PRIVILEGES ON craftsail_growth.* TO 'craftsail'@'%' ;
GRANT ALL PRIVILEGES ON craftsail_growth.* TO 'craftsail'@'localhost';
FLUSH PRIVILEGES;
