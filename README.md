# ProjectMoocow
bot to pin messages to another channel and publish to a website


How to start:

1. make a MySQL user and add their information to config.yml
1. run the following mysql code (PLEASE CHANGE THE DEFAULT PASSWORD): 
```
CREATE USER 'pinnerboi'@'localhost' IDENTIFIED BY 'p1nnedh1m!!1';
create database pinnerboibot;
use pinnerboibot;
CREATE TABLE pinnedmessages ( serverid CHAR(25), channelid CHAR(25), messageid CHAR(25));
GRANT ALL PRIVILEGES ON pinnerboibot.* TO 'pinnerboi'@'localhost';
```
1. also make sure you enable rich prescense and message content perms, the former is possibly not needed but the latter is needed 1. because I am protesting their new slash commands stupidity and 2. even if I did use slash commands, I still need message content to add to the pins

Also, there is some memory leak in here somewhere, I don't know from what or why, but it's there
