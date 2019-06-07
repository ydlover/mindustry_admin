支持如下功能
===========
* 1)游戏中通过聊天窗口发送命令，例如/gameover
* 2)基本权限控制，只有admin或者super admin才有命令执行权限
* 3)地图管理器，管理员可以通过web页面更换地图 
* 4)整点(每小时)自动备份功能 
* 5)投票功能，例如普通玩家可以通过/votetick hostx 1发起更换地图的投票
* 6)网络ban功能（默认不启用)，请通过修改config.ini中相应配置启用

使用方法
=========
* 1)将发布的zip解压到硬盘任意地方
* 2)将官方发布的server-release.jar也移动到该目录中
* 3)增加或修改admin.json文件中服务器的超级管理员名单(只需要配置name,id不用配置，用户第一次登录游戏时会自动记录)，当服务器启动时会将这些用户加载到系统中
* 4)启动对应操作系统的执行程序，例如 mindustry_admin_linux_386 -port 6567 -up 6569
* 5)启动参数说明:-port 服务器端口，默认6567，如果不需要修改可以不用输入
* 6)启动参数说明:-up 地图管理端口，默认6569，如果不需要修改可以不用输入


聊天室管理员命令帮助
====================
* 1)/maps 查看当前可用地图，地图前面的数字是ID
* 2)/hostx <map id> [mode] 使用ID换地图功能，换地图时会自动重启服务端
* 3)/save [slot] 当slot不输入时，自动保存为当前日时分/10对应的数字.注意保存时最好不要有没消灭的怪，否则load存档时可能将这些怪刷新到地图任意地方
* 4)/load slot 命令执行前检查slot是否非法，load时会自动重启服务端
* 5)/slots 查看当前服务器上可用存档
* 6)/show 查看服务器的管理员名单，默认普通用户可执行
 
Feture lists
============
* 1) In the game, commands are sent through chat windows, such as gameover [already supported]
* 2) Basic privilege control, only admin or superAdmin has command execution privilege [already supported]
* 3) Map Manager, which allows administrators to change maps through web pages [already supported]
* 4) Integer point (hourly) automatic backup function [already supported]
 
Installation
============
* 1) Unzip the published zip anywhere on the hard disk
* 2) Move the officially published server-release.jar to the directory
* 3) Modify the list of server administrators and super administrators in config.ini file, and load these users into the system when the server starts.
* 4) Start the execution program of the corresponding operating system, such as mindustry_admin_linux_386 -port 6567 -up 6569
* 5) Startup parameter description: - Port server port, default 6567, if you do not need to modify you can not enter
* 6) Startup parameter description: - up map management port, default 6569, if you do not need to modify you can not enter
 
Chat room command help
===================================
 * 1)/maps 
 to view the currently available map, the number in front of the map is ID
* 2)/hostx[mode] 
 Uses ID to change maps. When changing maps, the server will automatically restart.
* 3) /save [slot] 
 When slot is not input, it is automatically saved as the corresponding number of the current day time/10. Note that it is better not to have any enemy that have not been eradicated when saving, otherwise these monsters may be refreshed to any place on the map when loading archives.
* 4)/Load slot 
 Command checks whether the slot is illegal before execution, and automatically restarts the server when loading
 5)/slots 
 View the available archives on the current server. Note that if the archive version does not match, the map does not exist and other reasons may fail to load, the failure of loading needs to be handled manually.
* 6)/ShowAdmin
  View the server administrator list, default ordinary user execute
* 7)/vote [cmd] norm user vote in 60s
