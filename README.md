# ProjectMoocow
bot to pin messages to another channel and publish to a website


How to start:

1. make a MySQL user and add their information to config.yml
1. run the following mysql code (PLEASE CHANGE THE DEFAULT PASSWORD): ```
CREATE USER 'pinnerboi'@'localhost' IDENTIFIED BY 'p1nnedh1m!!1';
create database pinnerboibot;
use pinnerboibot;
CREATE TABLE pinnedmessages ( serverid CHAR(18), channelid CHAR(18), messageid CHAR(18));
GRANT ALL PRIVILEGES ON pinnerboibot.* TO 'pinnerboi'@'localhost';
```