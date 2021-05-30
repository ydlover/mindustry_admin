// aim
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type Aim struct {
	aim   *Aim
	mdt   *Mindustry
	cmdIn io.WriteCloser
}

func (this *Aim) init(mdt *Mindustry) {
	this.mdt = mdt
}

func (this *Aim) runJs(js string) {
	this.mdt.execCmd("js " + js)
}

/*
func (this *Aim) cmds(index int)string{
	init:=[...]string{
		"playerlist=[]",
		"Vars.netServer.admin.addChatFilter((player,text)=>{if(text.substr(0,1)!=\"/\"){a=text}else{cmd0(text,player);a=null}};return a)"}
	cmds:=[...]string{
		"test","p.sendMessage(\"test\")"}
	a:=len(cmds)/2
	mi:=len(init)+a
	if (index>mi){
		e:="EOF"
	}else{
		if(index<=len(init)){
			return init[index]
		}else{
			b:=index-len(init)*2
			c:=b+1
			d:=index-len(init)
			funct:="function "+cmds[b]+"(t,p){"+cmds[c]+"}"
			if (index==mi){
				cmdIf:="function cmd"+strconv.Itoa(d)+"(t,p){if(t.startWith(\""+cmds[b]+"\")){"+cmds[b]+"(t,p)}}"
			}else{
				cmdIf:="function cmd"+strconv.Itoa(d)+"(t,p){if(t.startWith(\""+cmds[b]+"\")){"+cmds[b]+"(t,p)}else{cmd"+strconv.Itoa(d+1)+"(t,p)}}"
			}
		}
		e:=funct+"\n"+cmdIf
	}
	return e
}
*/
func (this *Aim) save(data string, dataType string) {
	fileName := ""
	if dataType == "userinfo" {
		fileName = "Userinfo.bin"
	}
	if dataType == "config" {
		fileName = "Config.bin"
	}
	file, err := os.OpenFile("aim"+fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Println("open file aim"+fileName+" failed,err:", err)
		this.runJs("Call.sendMessage(\"[red]<!>open file aim" + fileName + " error!\")")
		return
	}
	this.runJs("Call.sendMessage(\"[green]file aim" + fileName + " saved!\")")
	defer file.Close()
	_, err = file.WriteString(data)
	if err != nil {
		fmt.Println("white file aim"+fileName+" failed,err:", err)
		this.runJs("Call.sendMessage(\"[red]<!>white file aim" + fileName + " error!\")")
		return
	}
	file.Sync()
	return
}
func (this *Aim) printEvent(data string) {
	a := strings.Split(data, " ")
	if a[0] == "playerJoin" {
		b := strings.TrimPrefix(data, "playerJoin ")
		c := strings.Split(b, "|-|")
		u := this.mdt.users[c[1]]
		this.mdt.offlineUser(u.Name, c[1])
		if u.IsSuperAdmin {
			this.mdt.execCmd("admin add " + c[1])
			this.mdt.onlineUser(c[0], c[1])
			this.mdt.onlineSuperAdmin(c[1])
		} else if u.IsAdmin {
			this.mdt.onlineUser(c[0], c[1])
			this.mdt.onlineAdmin(c[1])
		} else {
			this.mdt.onlineUser(c[0], c[1])
		}
	} else if a[0] == "runjs" {
		b := strings.TrimPrefix(data, "runjs ")
		c := strings.Split(b, "|||||")
		fmt.Printf("runjs: try{function result(){return " + c[0] + "};getP(\"" + c[1] + "\").sendMessage(\"Result: \"+result())}catch(err){getP(\"" + c[1] + "\").sendMessage(\"runERR: \"+err)}\n")
		data := []byte("js try{function result(){return " + c[0] + "};getP(\"" + c[1] + "\").sendMessage(\"Result: \"+result())}catch(err){getP(\"" + c[1] + "\").sendMessage(\"runERR: \"+err)}\n")
		this.mdt.cmdIn.Write(data)
	}
}

func (this *Aim) AimInit() {
	userinfo := "aim_userinfo=[]"
	config := "aim_config=[]"
	data := []byte("config desc off" + "\n")
	this.mdt.cmdIn.Write(data)
	js := []string{
		userinfo,
		config,
		"aim_enabled=true",
		"aim_canUseJs=false",
		"aim_disCmd={}",
		"aim_logLine=0",
		"aim_score=[]",
		"aim_unitOp={}",
		"aim_coreUnit=[]",
		"aim_score_name=[\"在线\",\"建造\",\"波次\",\"拆除\",\"采矿\",\"???\"]",
		"aim_score_toexp=[0.001,0.00125,0.00125,0.001,0.001,0.00125]",
		"aim_score_topoc=[0.002,0.003,0.025,0,0.0001,0.00125]",
		"aim_score_topwc=[0.001,0.0001,0.001,0.00025,0,0.00125]",
		"aim_score_maxsc=[1000,4000,1000,3000,2000,3000]",
		"aim_blocks=[];for(i=0;i<10000;i++){aim_blocks[i]=[]}",
		"try{Vars.netServer.admins.addChatFilter((player,message)=>{if(aim_enabled==false){return message}else{if(message.startsWith(\"*\")){return aim_cmdIf(player,message)}else{return message}}})}catch(e){print(e)}",
		"try{function aim_cmdIf(p,t){try{a=t.split(\" \");if(a[0]==\"*help\"){aim_help(p,a);return null}else if(a[0]==\"*spawn\"){aim_spawn(p,a);return null}else if(a[0]==\"*mono\"){aim_mono(p);return null}else if(a[0]==\"*team\"){aim_team(p,a);return null}else if(a[0]==\"*setblock\"){aim_setBlock(p,a);return null}else if(a[0]==\"*fill\"){aim_fill(p,a);return null}else if(a[0]==\"*js\"){aim_js(p,t);return null}else{return aim_cmdIf1(p,a)}}catch(e){print(e)}}}catch(e){print(e)}",
		"function aim_cmdIf1(p,a){try{if(a[0]==\"*ob\"){aim_ob(p);return null}else if(a[0]==\"*info\"){aim_info(p);return null}else if(a[0]==\"*core\"){aim_core(p);return null}else if(a[0]==\"*aimData\"){aim_aimData(p,a);return null}else if(a[0]==\"*poly\"){aim_poly(p);return null}else if(a[0]==\"*mega\"){aim_mega(p);return null}else if(a[0]==\"*mapinfo\"){aim_mapinfo(p);return null}else{return \"错误的指令:\"+a[0]}}catch(e){print(e)}}",
		"function aim_mono(p){try{if(aim_disCmd.mono!=undefined){p.sendMessage(\"指令 mono 被禁用!\")}else{if(aim_use(p,10,15)==false){p.sendMessage(\"point或power不足!\")}else{a=aim_gi(p);if(a.level>6){amou=6}else{amou=a.level};for(i=0;i<amou;i++){UnitTypes.mono.spawn(p.team(),p.x,p.y)};say(\"[aim] \"+p.name+\" 使用了技能 mono 生成了 \"+amou+\" 个 mono\")}}}catch(e){print(e)}}",
		"function aim_poly(p){try{if(aim_disCmd.poly!=undefined){p.sendMessage(\"指令 poly 被禁用!\")}else{a=aim_gi(p);if(a.level<3){p.sendMessage(\"等级不足，需要3\")}else{if(aim_use(p,45,55)==false){p.sendMessage(\"point或power不足!\")}else{a=aim_gi(p);if(a.level>6){amou=4}else{amou=a.level/2+1};for(i=0;i<parseInt(amou);i++){UnitTypes.poly.spawn(p.team(),p.x,p.y)};say(\"[aim] \"+p.name+\" 使用了技能 poly 生成了 \"+amou+\" 个 poly\")}}}}catch(e){print(e)}}",
		"function aim_mega(p){try{if(aim_disCmd.mega!=undefined){p.sendMessage(\"指令 mega 被禁用!\")}else{a=aim_gi(p);if(a.level<4){p.sendMessage(\"等级不足，需要4\")}else{if(aim_use(p,75,65)==false){p.sendMessage(\"point或power不足!\")}else{a=aim_gi(p);if(a.level>8){amou=3}else{amou=a.level/4+1};for(i=0;i<parseInt(amou);i++){UnitTypes.mega.spawn(p.team(),p.x,p.y)};say(\"[aim] \"+p.name+\" 使用了技能 mega 生成了 \"+amou+\" 个 mega\")}}}}catch(e){print(e)}}",
		"function aim_core(p){try{if(aim_disCmd.core!=undefined){p.sendMessage(\"指令 core 被禁用!\")}else{a=aim_gi(p);if(a.level<3){p.sendMessage(\"等级不足，需要3\")}else{if(aim_use(p,55,65)==false){p.sendMessage(\"point或power不足!\")}else{UnitTypes.oct.spawn(p.team(),p.x,p.y).maxHealth=1;Vars.world.tile(p.tileX(),p.tileY()).setNet(Blocks.coreShard,p.team(),0);say(\"[aim] \"+p.name+\" 使用了技能 core 召唤了核心\")}}}}catch(e){print(e)}}",
		"function say(a){try{Call.sendMessage(a)}catch(e){print(e)}}",
		"function getP(n){try{a=null;Groups.player.each(b=>{if(b.name==n){a=b}});return a}catch(e){print(e)}}",
		"function getPuuid(n){try{a=null;Groups.player.each(b=>{if(b.uuid()==n){a=b}});return a}catch(e){print(e)}}",
		"function aim_js(p,m){try{if(aim_canUseJs==false){p.sendMessage(\"js被禁用\")}else{if(p.admin==false){p.sendMessage(\"你不是管理员!\")}else{print(\"runjs \"+m.substr(4,m.length)+\"|||||\"+p.name)}}}catch(e){print(e)}}",
		"function aim_team(p,a){try{if(p.admin==true){if(isNaN(a[1])){p.sendMessage(\"请使用数字队伍id\")}else{if(a.length==2){p.team(Team.get(a[1]));say(\"[aim] 管理员 \"+p.name+\" 修改了 \"+p.name+\" 的队伍为 \"+Team.get(a[1]))}else{if(a.length>2){pp=getP(a[2]);if(pp==null){p.sendMessage(\"找不到玩家!\");}else{pp.team(Team.get(a[1]));say(\"[aim] 管理员 \"+p.name+\" 修改了 \"+pp.name+\" 的队伍为 \"+Team.get(a[1]))}}else{p.sendMessage(\"参数错误\")}}}}else{p.sendMessage(\"你不是管理员!\");}}catch(e){print(e)}}",
		"function aim_help(p,t){try{Call.infoMessage(p.con,\"*help 帮助\\n*spawn <unit> [teamid] [amount] 生成单位 [red]Admin Only[white]\\n*team <teamid> [player] 更改队伍 [red]Admin Only[white]\\n*aimData <type> [value] [player] 修改玩家的aim数据 [red]Admin Only[white]\\n*setblock <x> <y> <blockName> [teamid] [rotation] 设置方块 [red]Admin Only\\n[white]*fill <x1> <y1> <x2> <y2> <blockName> [teamid] [rotation] 填充方块 [red]Admin Only\\n[white]*info 用户信息\\n*ob 观察者\\n*mono 无情的采矿机\\n*core 生成一个小型核心+一个1血oct\\n*poly 自动重建机\\n*mega 移动的修复器\\n*mapinfo 地图信息\")}catch(e){print(e)}}",
		"function aim_mapinfo(p){try{m=Vars.state.map;Call.infoMessage(p.con,\"---地图信息---\\n地图名:\"+m.name()+\"\\n作者:\"+m.author()+\" 版本:\"+m.version+\"\\n介绍:\"+m.description())}catch(e){print(e)}}",
		"function aim_spawn(p,t){try{if(p.admin==true){if(t.length>1){if(t.length>2){if(isNaN(a[2])){p.sendMessage(\"请使用数字队伍id\");return}else{team=Team.get(a[2])}}else{team=p.team()};if(t.length>3){amou=a[3]}else{amou=1};u=Vars.content.getByName(ContentType.unit,t[1]);if(u==null){p.sendMessage(\"错误名称!\")}else{for(i=0;i<amou;i++){u.spawn(team,p.x,p.y)};say(\"[aim] 管理员 \"+p.name+\" 召唤了 \"+amou+\" 个 \"+t[1]+\" ,队伍为 \"+team)}}else{p.sendMessage(\"参数错误\")}}else{p.sendMessage(\"你不是管理员!\")}}catch(e){print(e)}}",
		"function aim_setBlock(p,t){try{if(p.admin==true){if(t.length>3){if(t.length>4){team=Team.get(a[4])}else{team=p.team()};if(t.length>5){r=a[5]}else{r=0};b=Vars.content.getByName(ContentType.block,t[3]);if(b==null){p.sendMessage(\"错误名称!\")}else{Vars.world.tile(t[1],t[2]).setNet(b,team,r);say(\"[aim] 管理员 \"+p.name+\" 设置了 \"+t[1]+\" \"+t[2]+\" 处的方块为 \"+t[3]+\" ,队伍为 \"+team)}}else{p.sendMessage(\"参数错误\")}}else{p.sendMessage(\"你不是管理员!\")}}catch(e){print(e)}}",
		"function aim_fill(p,t){try{if(p.admin==false){p.sendMessage(\"你不是管理员!\")}else{if(t.lenght>5||t[1]>t[3]||t[2]>t[4]){p.sendMessage(\"参数错误!\")}else{if(t.length<7){te=p.team()}else{if(isNaN(a[6])){p.sendMessage(\"请使用数字队伍id\");return}else{te=Team.get(a[6])}};if(t.length<8){r=0}else{r=t[7]};b=Vars.content.getByName(ContentType.block,t[5]);if(b==null){p.sendMessage(\"错误名称!\")}else{for(x=t[1];x<t[3];x=x+b.size){for(y=t[2];y<t[4];y=y+b.size){if(Vars.world.tile(x,y)==null){break}else{Vars.world.tile(x,y).setNet(b,te,r)}}};say(\"[aim] 管理员 \"+p.name+\" 填充了从 \"+t[1]+\" \"+t[2]+\" 到 \"+t[3]+\" \"+t[4]+\" 的方块为 \"+t[5]+\" ,队伍为 \"+te)}}}}catch(e){print(e)}}",
		"function aim_tick(){try{Groups.player.each(p=>{nt=Date.now()/1000;a=aim_gi(p);b=nt-a.lastupdate;ap=1/a.fixpoint*b;apw=1/a.fixpower*b;full=0;if(apw+a.power>a.maxpower){apw=a.maxpower-a.power;full=1};aim_ap(p,ap);aim_apw(p,apw);a=aim_gi(p);if(full==1){a.canuse=true};a.lastupdate=Date.now()/1000;aim_si(p,a)})}catch(e){print(e)}}",
		"function aim_saveData(){try{ud=\"userinfo [\";for(a in aim_userinfo){c=\"{\";for(b in aim_userinfo[a]){if(isNaN(aim_userinfo[a][b])==false){d=parseInt(aim_userinfo[a][b]*1000)/1000;c=c+b+\":\"+d+\",\"}else{d=aim_userinfo[a][b];c=c+b+\":\\\"\"+d+\"\\\",\"}};c=c.substr(0,c.length-1)+\"},\";ud=ud+c};ud=ud.substr(0,ud.length-1)+\"]\";print(ud)}catch(e){print(e)}}",
		"function aim_gi(p){try{found=aim_userinfo.map((item) => item.uuid).indexOf(p.uuid());if(found==-1){return {uuid:p.uuid(),point:25,maxpoint:25,fixpoint:120,power:20,maxpower:20,fixpower:60,canuse:true,lastupdate:Date.now()/1000,level:1,exp:0,totalscore:0,isafk:6}}else{return aim_userinfo[found]}}catch(e){print(e)}}",
		"function aim_si(p,a){try{f=aim_userinfo.map((item) => item.uuid).indexOf(p.uuid());if(f==-1){l=aim_userinfo.length;aim_userinfo[l]=a}else{aim_userinfo[f]=a}}catch(e){print(e)}}",
		"function aim_apw(p,a){try{b=aim_gi(p);b.power=b.power+a;if(b.canuse==false){return false};if(b.power<0){b.canuse=false};aim_si(p,b);return true}catch(e){print(e)}}",
		"function aim_ap(p,a){try{b=aim_gi(p);if(a<0){if(b.point+a<0){return false}else{b.point=b.point+a;aim_si(p,b);return true}}else{b.point=b.point+a;if(b.point>b.maxpoint){b.point=b.maxpoint};aim_si(p,b);return true}}catch(e){print(e)}}",
		"function aim_use(p,po,pw){try{a=aim_gi(p);if(a.point<po||a.canuse==false){return false}else{aim_ap(p,0-po);aim_apw(p,0-pw);return true}}catch(e){print(e)}}",
		"function aim_info(p){try{a=aim_gi(p);if(a.canuse==false){add=\"\\n[yellow]<!>Power过载![white]\"}else{add=\"\"};Call.infoMessage(p.con,\"用户信息\\nusid:\"+p.usid()+\"\\nuuid:\"+p.uuid()+\"\\npoint:\"+parseInt(a.point*1000)/1000+\"/\"+parseInt(a.maxpoint*1000)/1000+\" \"+parseInt(a.fixpoint*1000)/1000+\"秒/单位\\npower:\"+parseInt(a.power*1000)/1000+\"/\"+parseInt(a.maxpower*1000)/1000+\" \"+parseInt(a.fixpower*1000)/1000+\"秒/单位\\n\"+add+\"\\n总分数:\"+parseInt(a.totalscore*1000)/1000+\"\\nLV:\"+a.level+\"   EXP:\"+parseInt(a.exp*1000)/1000+\"/\"+a.level+\"\\nAim beta1.0.3 by awa(3328796027)\")}catch(e){print(e)}}",
		"function aim_ob(p){try{p.team(Team.get(255));p.unit(UnitTypes.gamma.spawn(p.team(),p.x,p.y));say(\"[aim] \"+p.name+\" 选择成为了观察者\")}catch(e){print(e)}}",
		"function aim_exp_add(p,e){try{a=aim_gi(p);a.exp=a.exp+e;if(a.exp>=a.level){a.exp=0;a.level++};aim_si(p,a)}catch(e){print(e)}}",
		"function aim_aimData(p,a){try{if(p.admin==false){p.sendMessage(\"你不是管理员!\")}else{if(a.length<2){p.sendMessage(\"参数错误\")}else{if(a.length>=4){pp=p;p=getP(a[3]);if(p==null){pp.sendMessage(\"找不到玩家!\");return }else{pp=p};if(a.length==2){b=aim_gi(p);pp.sendMessage(b[a[1]])}else{b=aim_gi(p);if(b[a[1]]==undefined){pp.sendMessage(\"不存在的变量!\")}else{b[a[1]]=a[2];aim_si(p,b)}}}}}}catch(e){print(e)}}",
		"function aim_score_add(p,s,t){try{t=t+3;a=aim_score.map((a)=>a[0]).indexOf(p.uuid());if(a==-1){b=[p.uuid(),p.name,p.team(),0,0,0,0,0,0];b[t]=s;aim_score[aim_score.length]=b}else{aim_score[a][t]=aim_score[a][t]+s}}catch(e){print(e)}}",
		"function aim_score_give(win){try{for(a in aim_score){exp=[];apoc=[];apwc=[];for(i=0;i<6;i++){b=aim_score[a][i+3];if(b>aim_score_maxsc[i]){b=aim_score_maxsc[i]};exp[i]=parseInt(b*aim_score_toexp[i]*1000)/1000;apoc[i]=parseInt(b*aim_score_topoc[i]*1000)/1000;apwc[i]=parseInt(b*aim_score_topwc[i]*1000)/1000};if(aim_score[a][2]==win||aim_if_winWave()){win=\"[green]Win!--*1.25--*1.25--*1.25--*1.25[white]\";w=1.25}else{win=\"\";w=1};y=aim_userinfo.map((item)=>item.uuid).indexOf(aim_score[a][0]);data=aim_userinfo[y];x=(data.level-1)*0.01;w=w+x;tsc=0;texp=0;tpoc=0;tpwc=0;mess=\"------结果------\\n名称--分数--exp--point 存储--power存储\";for(i=0;i<6;i++){mess=mess+\"\\n\"+aim_score_name[i]+\"--\"+aim_score[a][i+3]+\"--\"+exp[i]+\"--\"+apoc[i]+\"--\"+apwc[i];tsc=aim_score[a][3+i]*w+tsc;texp=exp[i]*w+texp;tpoc=apoc[i]*w+tpoc;tpwc=apwc[i]*w+tpwc};addtext=\"\";px=(data.level-1)*0.01+1;if(aim_if_issandbox()==true){addtext=addtext+\"\\n[yellow]<!>沙盒模式，无法获得分数\"};if(aim_if_giveexp()==false){addtext=addtext+\"[yellow]\\n<!>地图不符合获得经验的条件\"};mess=mess+win+\"\\nlevel.\"+data.level+\"--*\"+px+\"--*\"+px+\"--*\"+px+\"--*\"+px+\"\\n-------------------\\n总和--\"+parseInt(tsc*1000)/1000+\"--\"+parseInt(texp*1000)/1000+\"--\"+parseInt(tpoc*1000)/1000+\"--\"+parseInt(tpwc*1000)/1000+\"\\n\"+addtext;pid=aim_score[a][0];if(getPuuid(pid)!=null){Call.infoMessage(getPuuid(pid).con,mess)};data=aim_userinfo[y];if(aim_if_giveexp()==true){data.exp=data.exp+texp;data.maxpoint=data.maxpoint+tpoc;data.maxpower=data.maxpower+tpwc};if(aim_if_issandbox()==false){data.totalscore=data.totalscore+tsc};aim_userinfo[y]=data};aim_score=[]}catch(e){print(e)}}",
		"function aim_log_spawn(x,y,pn,t){try{aim_logs[-1]=\"[\"+phaseInt(x)+\" \"+phaseInt(y)+\"][\"+pn+\"]\"+t;for(i=-1;i<10;i++){aim_logs[i+1]=aim_logs[i]}}catch(e){print(e)}}",
		"function aim_if_issandbox(){try{if(Vars.state.rules.infiniteResources==true){return true}else{return false}}catch(e){print(e)}}",
		"function aim_if_giveexp(){try{if(aim_if_issandbox()==true){return false}else if(Vars.state.rules.buildCostMultiplier<=0.98){return false}else if(Vars.state.rules.deconstructRefundMultiplier>=1.01){return false}else{return true}}catch(e){print(e)}}",
		"function aim_if_winWave(){try{if(Vars.state.rules.pvp!=true&&Vars.state.wave>=Vars.state.rules.winWave&&Vars.state.rules.winWave>0){return true}else{return false}}catch(e){print(e)}}",
		"function aim_init_1(){Timer.schedule((()=>{try{aim_tick();Call.infoPopup(\"欢迎来到此服务器\\n使用*help查看Aim帮助\\n使用!help查看\\nmindustry_admin帮助\\n使用*mapinfo\\n查看地图信息\",5,Align.topLeft,150,0,0,0);Groups.unit.each(u=>{if(aim_unitOp[u.type]==1){Call.infoToast(\"[red]unit \"+u.type+\" is disabled.\",5);Timer.schedule((()=>{u.kill()}),0.5)}else if(aim_unitOp[u.type]!=undefined){Timer.schedule((()=>{aim_unit_replace(u.x,u.y,u.team,u.player,aim_unitOp[u.type]);u.kill()}),0.5)}});for(a in aim_coreUnit){u=aim_coreUnit[a];if(u!=null&&u.player==null){u.kill();aim_coreUnit[a]=null}}}catch(e){print(e)}}),5,5)}",
		"function aim_init_2(){Timer.schedule((()=>{try{Groups.player.each(p=>{aim_score_add(p,1,0);if(p.unit().mineTile!=null){aim_score_add(p,1,4)}})}catch(e){print(e)}}),20,20)}",
		"function aim_init_3(){Timer.schedule((()=>{try{aim_saveData()}catch(e){print(e)}}),600,600)}",
		"function aim_init_4(){Events.on(BlockBuildEndEvent,((a)=>{try{if(a.unit.player!=null){try{b=a.tile.build.getDisplayName();aim_blocks[parseInt(a.tile.x/8)][parseInt(a.tile.y/8)]=b;aim_score_add(a.unit.player,0.125,1)}catch(e){aim_score_add(a.unit.player,0.125,3);aim_blocks[parseInt(a.tile.x/8)][parseInt(a.tile.y/8)]=null}}}catch(e){print(e)}}))}",
		"function aim_init_5(){Events.on(WaveEvent,(()=>{try{Groups.player.each(p=>{aim_score_add(p,1,2)})}catch(e){print(e)}}))}",
		"function aim_init_6(){Events.on(GameOverEvent,((a)=>{try{aim_score_give(a.winner);aim_blocks=[];for(i=0;i<10000;i++){aim_blocks[i]=[]};Timer.schedule((()=>{aim_event_loadDisCmd()}),12)}catch(e){print(e)}}))}",
		"function aim_init_7(){Events.on(PlayerJoin,((a)=>{try{p=a.player;data=aim_gi(p);aim_si(p,data);aim_exp_add(p,0);p.name=\"[LV.\"+data.level+\"]|\"+p.name;print(\"playerJoin \"+p.name+\"|-|\"+p.uuid())}catch(e){print(e)}}))}",
		"function aim_init_8(){Events.on(UnitUnloadEvent,((a)=>{try{if(aim_unitOp[a.unit.type]==1){Call.infoToast(\"[red]unit \"+a.unit.type+\" is disabled.\",5);Timer.schedule((()=>{a.unit.kill()}),0.5)}else if(aim_unitOp[a.unit.type]!=undefined){Timer.schedule((()=>{aim_unit_replace(a.unit.x,a.unit.y,a.unit.team,a.unit.player,aim_unitOp[a.unit.type]);a.unit.kill()}),0.5)}}catch(e){print(e)}}))}",
		"function aim_unit_replace(x,y,team,player,to){try{if(player!=null){player.unit(to.spawn(player.team(),player.x,player.y));aim_coreUnit[aim_coreUnit.length]=player.unit()}else{to.spawn(team,x,y)}}catch(e){print(e)}}",
		"function aim_mapTag_varVaule(t){try{Vars.state.rules.winWave=0;a=t.split(\"=\");if(a.length>1){if(a[0]==\"startUnit\"){aim_mapTag_startUnit(a[1])}else if(a[0]==\"winWave\"){aim_mapTag_winWave(a[1])}}}catch(e){print(e)}}",
		"function aim_mapTag_winWave(t){try{if(!isNaN(t)){Vars.state.rules.winWave=t}}catch(e){print(e)}}",
		"function aim_mapTag_unit(t){try{te=t.substr(1,t.length);if(t.startsWith(\"!\")){u=Vars.content.getByName(ContentType.unit,te);if(u!=null){aim_unitOp[u]=1}}else if(t.startsWith(\"#\")){me=te.split(\"->\");u=Vars.content.getByName(ContentType.unit,me[0]);tou=Vars.content.getByName(ContentType.unit,me[1]);if(u!=null||tou!=null){aim_unitOp[u]=tou}}}catch(e){print(e)}}",
		"function aim_mapTag_startUnit(t){try{a=t.split(\"/\");if(a.length>3){x=a[0]*8;y=a[1]*8;if(!isNaN(a[2])){te=Team.get(a[2]);units=a[3].split(\";\");for(u in units){un=units[u].split(\",\");uty=Vars.content.getByName(ContentType.unit,un[0]);if(uty!=null){for(i=0;i<un[1];i++){uty.spawn(te,x,y)}}}}}}catch(e){print(e)}}",
		"function aim_event_loadDisCmd(){try{aim_coreUnit=[];aim_disCmd={};aim_unitOp={};dp=Vars.state.map.description().split(\"[\");for(a in dp){t=dp[a];if(t.startsWith(\"!\")){t=t.split(\"]\");aim_disCmd[t[0].substr(1,t[0].length)]=1}else if(t.startsWith(\"@\")){t=t.split(\"]\");aim_mapTag_varVaule(t[0].substr(1,t[0].length))}else if(t.startsWith(\"u\")){t=t.split(\"]\");aim_mapTag_unit(t[0].substr(1,t[0].length))}}}catch(e){print(e)}}",
		"function aim_admin_host(mapname,gamemode){try{map=Vars.maps.byName(mapname);if(map==null){return false};if(gamemode==\"\"){rule=map.rules()}else{gamem=aim_admin_getGamemode(gamemode);if(gamem==null){return false};rule=map.applyRules(gamem)};aim_score_give(\"\");rel=new WorldReloader();rel.begin();Vars.world.loadMap(map,rule);Vars.state.rules=rule;Vars.logic.play();rel.end();aim_event_loadDisCmd();return true}catch(e){print(e)}}",
		"function aim_admin_hostx(mapid,gamemode){try{map=aim_admin_getMap(mapid);if(map==null){return false};if(gamemode==\"\"){rule=map.rules()}else{gamem=aim_admin_getGamemode(gamemode);if(gamem==null){return false};rule=map.applyRules(gamem)};aim_score_give(\"\");rel=new WorldReloader();rel.begin();Vars.world.loadMap(map,rule);Vars.state.rules=rule;Vars.logic.play();rel.end();aim_event_loadDisCmd();;return true}catch(e){print(e)}}",
		"function aim_admin_getMap(id){try{a=null;b=1;Vars.maps.all().each((m)=>{if(b==id){a=m};b++});return a}catch(e){print(e)}}",
		"function aim_admin_getGamemode(t){try{if(t==\"s\"||t==\"survival\"){return Gamemode.survival}else if(t==\"a\"||t==\"attack\"){return Gamemode.attack}else if(t==\"p\"||t==\"pvp\"){return Gamemode.pvp}else if(t==\"c\"||t==\"sandbox\"){return Gamemode.sandbox}else{return null}}catch(e){print(e)}}",
		"try{try{aim_runed}catch(e){aim_runed=1;aim_init_1();aim_init_2();aim_init_3();aim_init_4();aim_init_5();aim_init_6();aim_init_7();aim_init_8()}}catch(e){print(e)}",
		"try{try{if(isHost==null){aim_event_loadDisCmd();isHost=1}}catch(e){Timer.schedule((()=>{aim_event_loadDisCmd()}),12);isHost=1}}catch(e){print(e)}"}
	file, err := ioutil.ReadFile("aimUserinfo.bin")
	if err != nil {
		fmt.Println("\nopen file aimUserinfo.bin failed,err:", err)
		data := []byte("config desc [red]<!>open file aimUserinfo.bin error!\n")
		this.mdt.cmdIn.Write(data)
		this.loadJs(js)
		return
	}
	userinfo = "aim_userinfo=" + string(file)
	js[0] = userinfo
	/*
		file, err = ioutil.ReadFile("aimConfig.bin")
		if err != nil {
			fmt.Println("\nopen file aimConfig.bin failed,err:", err)
			data := []byte("config desc [red]<!>open file aimConfig.bin error!\n")
			this.mdt.cmdIn.Write(data)
			this.loadJs(js)
			return
		}
		config = "aim_config=" + string(file)
		js[1] = config
	*/
	this.loadJs(js)
	/*
		cmd:=""
		a:=0
		for cmd!="EOF"{
			cmd=this.cmds(a)
			data := []byte("js " + cmd + "\n")
			this.mdt.cmdIn.Write(data)
			a=a++
		}
	*/
}

func (this *Aim) loadJs(js []string) {
	fmt.Printf("\n")
	for i := 0; i < len(js); i++ {
		data := []byte("js " + js[i] + "\n")
		this.mdt.cmdIn.Write(data)
		fmt.Printf("[INFO]Aim runjs:" + js[i] + "\n")
	}
	/*
		cmd:=""
		a:=0
		for cmd!="EOF"{
			cmd=this.cmds(a)
			data := []byte("js " + cmd + "\n")
			this.mdt.cmdIn.Write(data)
			a=a++
		}
	*/
}
