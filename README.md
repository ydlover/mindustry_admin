![效果展示](https://github.com/ydlover/mindustry_admin/blob/master/web/demo_hostx.jpg?raw=true)
==========

支持如下功能
===========
* 1)游戏中通过聊天窗口发送命令，例如/gameover  
* 2)基本权限控制，只有admin或者super admin才有命令执行权限  
* 3)地图管理器，管理员可以通过web页面更换地图   
* 4)整点(每小时)自动备份功能   
* 5)投票功能，例如普通玩家可以通过/votetick hostx 1发起更换地图的投票  

使用方法
=========
## 1)将发布的zip解压到硬盘任意地方
## 2)将官方发布的server-release.jar也移动到该目录中
## 3)增加或修改admin.json文件中服务器的超级管理员名单(只需要配置name,id不用配置，用户第一次登录游戏时会自动记录)，当服务器启动时会将这些用户加载到系统中  
注意：  
1. 只需要配置name,不需要配置id，id将在该玩家登录时自动绑定  
2. 普通管理员建议不要修改admin.json来配置，因为修改不正确会导致所有管理员配置都不生效，请超级管理员在游戏中通过/admin xxx来配置  
2. 游戏运行中禁止手动修改admin.json,请先在控制台执行exit命令退出管理程序后再修改该文件

## 4)修改config.ini中相关配置
    mindustryPort mindustry启动端口，默认6567  
    mapMangePort  地图管理端口，默认6569  
## 5)启动对应操作系统的执行程序，例如 mindustry_admin_linux_386


聊天室管理员命令帮助
====================
* 1)/maps 查看当前可用地图，地图前面的数字是ID
* 2)/hostx <map id> [mode] 使用ID换地图功能，换地图时会自动重启服务端
* 3)/save [slot] 当slot不输入时，自动保存为当前日时分/10对应的数字.注意保存时最好不要有没消灭的怪，否则load存档时可能将这些怪刷新到地图任意地方
* 4)/load slot 命令执行前检查slot是否非法，load时会自动重启服务端
* 5)/slots 查看当前服务器上可用存档
* 6)/show 查看服务器的管理员名单，默认普通用户可执行

Web api
===============
* [get]黑名单列表:/blacklist
* [get]解封blacklist?unban=uuid
* [get]查看管理员列表/admins
* [get]超管删除管理员/admins?rmv=申请名
* [get]查看申请列表/signList
* [get]超管同意申请/signList?add=申请名
* [get]超管拒绝/signList?deny=申请名
* [get]地图列表/maps
* [get]地图下载/maps?download=文件名
* [get]地图删除/maps?delete=文件名
* [post]地图上传/maps,字段名file
* /mods(同地图)
* /plugins(同地图)
* [post]登录/login?username=xx&passwd=yy
* [post]注册管理员/sign?username=xx&passwd=yy&gamename=游戏里面昵称&contact=联系方式
* [post]修改密码/modifyPasswd?passwd=新密码
* [get]重置uuid，/resetUuid
* [get]重置uuid同时修改游戏里面的名字，/resetUuid?gamename=xxx

注意事项
============
会话保持(10分钟不操作会被下线):除sign和login外都必须带username和sessionid

回应只有三种:
============
* 1)maps/blacklist/admins这种回应列表json格式
* 2)login:{"result":"admin/sop/其它错误信息，可以用来显示在app","session":"11111"}
* 3)其它操作:{"result":"succ/其它错误信息，可以用来显示在app

Feture lists
=================
* 1) In the game, commands are sent through chat windows, such as gameover [already supported]
* 2) Basic privilege control, only admin or superAdmin has command execution privilege [already supported]
* 3) Map Manager, which allows administrators to change maps through web pages [already supported]
* 4) Integer point (hourly) automatic backup function [already supported]
 
Installation
============
## 1) Unzip the published zip anywhere on the hard disk
## 2) Move the officially published server-release.jar to the directory
## 3) Modify config.ini
     mindustryPort mindustry boot port, default 6567
     mapMangePort map management port, default 6569
## 4) Configure super administrator
1. Only need to configure name, no need to configure id, id will be automatically bound when the player logs in.  
2. Ordinary administrators do not recommend modifying admin.json to configure, because incorrect modification will cause all administrator configurations to take effect. Please super administrator to configure via /admin xxx in the game.  
3. It is forbidden to manually modify admin.json during game running. Please execute the exit command in the console to exit the management program and then modify the file.  
4. Start the execution program of the corresponding operating system, such as mindustry_admin_linux_386
 
in-game command help
===================================
 * 1)/maps 
 to view the currently available map, the number in front of the map is ID  
* 2)/hostx [id] <mode> 
 Uses ID to change maps. When changing maps, the server will automatically restart.  
* 3)/save [slot] 
 When slot is not input, it is automatically saved as the corresponding number of the current day time/10. Note that it is better not to have any enemy that have not been eradicated when saving, otherwise these monsters may be refreshed to any place on the map when loading archives.  
* 4)/Load slot 
 Command checks whether the slot is illegal before execution, and automatically restarts the server when loading  
* 5)/slots   
 View the available archives on the current server. Note that if the archive version does not match, the map does not exist and other reasons may fail to load, the failure of loading needs to be handled manually.  
* 6)/admins  
  View the server administrator list, default ordinary user execute  
* 7)/vote [cmd] norm user vote in 60s  
