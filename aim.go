package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Aim struct {
	Uuid           string  `json:"uuid"`
	Point          float64 `json:"point"`
	MaxPoint       float64 `json:"maxPoint"`
	Power          float64 `json:"power"`
	MaxPower       float64 `json:"maxPower"`
	LastUpdateTime int64   `json:"lastTickTime"`
	P_tTime        int64   `json:"p_tTime"`
	Pw_tTime       int64   `json:"pw_tTime"`
	CanUseAim      bool    `json:"canUseAim"`
}

type RandData struct {
	MinPoint       float64
	MaxPoint       float64
	MinUsePower    float64
	AddUsePower    float64
	Succ           int64
	AddSucc        int64
	MinBuildSpeed  float64
	MaxBuildSpeed  float64
	AddBuildSpeed  float64
	MinMoveSpeed   float64
	MaxMoveSpeed   float64
	AddMoveSpeed   float64
	MinItemCaption float64
	MaxItemCaption float64
	AddItemCaption float64
	MinMineSpeed   float64
	MaxMineSpeed   float64
	AddMineSpeed   float64
	MinHealth      float64
	MaxHealth      float64
	AddHealth      float64
	MinAmmoCaption float64
	MaxAmmoCaption float64
	AddAmmoCaption float64
	MinAmount      float64
	MaxAmount      float64
	AddAmount      float64
}

var jsdata string
var over bool

type AimInfo struct {
	AimUserInfo map[string][]Aim `json:"aim_lst"`
}
type AimManager struct {
	mdt        *Mindustry
	aimInfo    *AimInfo
	isNeedSave bool
}

var AIM_FILE_NAME string = "aim.json"

func (this *AimManager) loadAimInfo() {
	if !this.isNeedSave {
		return
	}
	data, err := ioutil.ReadFile(AIM_FILE_NAME)
	if err != nil {
		log.Printf("[ERR]Not found aim.json!\n")
		return
	}
	err = json.Unmarshal(data, this.aimInfo)
	if err != nil {
		log.Printf("[ERR]Load history fail:%s,err:%v!\n", AIM_FILE_NAME, err)
		return
	}
	this.isNeedSave = false
}

func (this *AimManager) saveAimInfo() {
	data, err := json.MarshalIndent(this.aimInfo, "", "    ")
	if err != nil {
		log.Println("[ERR]writeAdminCfg fail:", err)
		return
	}
	WriteConfig(AIM_FILE_NAME, data)
	log.Println("[INFO]aim is saved")
	this.isNeedSave = true
}
func (this *AimManager) tenMinProc() {
	this.saveAimInfo()
}

func (this *AimManager) init(mdt *Mindustry) {
	this.mdt = mdt
	this.aimInfo = new(AimInfo)
	this.aimInfo.AimUserInfo = make(map[string][]Aim, 0)
	this.loadAimInfo()
	over = true
}

func (this *AimManager) runjs(js string) string {
	this.mdt.execCmd("js " + js)
	return ""
}

func (this *AimManager) jsSay(message string) string {
	this.runjs("Call.sendMessage(\"" + message + "\")")
	return ""
}

// 如果要使用上面的函数使用this.aim.sayJs(xxx)

func (this *AimManager) jsdataProc(jsdataa string) bool {
	jsdata = jsdataa
	return true
}

func (this *AimManager) appendAimInfo(uuid string, data *Aim) bool {
	_, has := this.aimInfo.AimUserInfo[uuid]
	if !has {
		this.aimInfo.AimUserInfo[uuid] = make([]Aim, 0)
	}
	this.aimInfo.AimUserInfo[uuid] = append(this.aimInfo.AimUserInfo[uuid], *data)
	this.isNeedSave = true

	return true
}

func (this *AimManager) getAimInfo(uuid string) (ret []Aim) {
	data, has := this.aimInfo.AimUserInfo[uuid]
	if has {
		ret = append(ret, data...)
		return ret
	} else {
		return nil
	}
}

func (this *AimManager) max(data1 float64, data2 float64) float64 {
	if data1 >= data2 {
		return data1
	} else {
		return data2
	}
}

func (this *AimManager) min(data1 float64, data2 float64) float64 {
	if data2 >= data1 {
		return data1
	} else {
		return data2
	}
}

func (this *AimManager) aimInfoTick(uuid string) bool {
	//this.jsSay("tick1")
	_, has := this.aimInfo.AimUserInfo[uuid]
	has = has
	if !has {
		//this.jsSay("tick2")
		aim := new(Aim)
		aim.MaxPoint = 35.0
		aim.MaxPower = 35.0
		aim.P_tTime = 120
		aim.Pw_tTime = 60
		aim.CanUseAim = true
		this.appendAimInfo(uuid, aim)
	}
	for index, _ := range this.aimInfo.AimUserInfo[uuid] {
		aim := &(this.aimInfo.AimUserInfo[uuid][index])
		//this.jsSay("tick3")
		point := aim.Point
		maxPoint := aim.MaxPoint
		power := aim.Power
		maxPower := aim.MaxPower
		lastTime := aim.LastUpdateTime
		p_tTime := aim.P_tTime
		pw_tTime := aim.Pw_tTime
		nowTime := time.Now().UTC().Unix()
		a := nowTime - lastTime
		bp := float64(a / p_tTime)
		bpw := float64(a / pw_tTime)
		point = point + bp
		power = power + bpw
		this.jsSay(strconv.FormatFloat(point, 'f', -1, 64))
		this.jsSay(strconv.FormatFloat(power, 'f', -1, 64))
		this.jsSay(strconv.FormatInt(nowTime, 10))
		if power >= maxPower {
			power = maxPower
		}
		if point >= maxPoint {
			point = maxPoint
		}
		aim.Point = point
		aim.Power = power
		aim.LastUpdateTime = nowTime
		if power == maxPower {
			aim.CanUseAim = true
		}
	}
	return true
}
func (this *AimManager) randData(usePoint string, config RandData) (string, string, string, float64, float64) {
	c, _ := strconv.ParseFloat(usePoint, 64)
	//if err != nil {
	//	over = true
	//	return "", "", "", 0.0, 0.0
	//}
	up := this.min(c, config.MaxPoint)
	up1 := this.max(up, config.MinPoint)
	up = up1 - config.MinPoint
	usePower := config.AddUsePower*up + config.MinUsePower
	rand.Seed(time.Now().UnixNano())
	e := strconv.FormatInt(config.AddSucc, 10)
	d, _ := strconv.ParseFloat(e, 64)
	//if err != nil {
	//	over = true
	//	return "", "", "", 0.0, 0.0
	//}
	succ1 := up * d
	succ1 = succ1 + 100.0
	b := strconv.FormatFloat(succ1, 'f', -1, 64)
	//succ2, _ := strconv.ParseInt(b, 10, 64)
	succ8, _ := strconv.Atoi(b)
	//if err != nil {
	//	over = true
	//	return "", "", "", 0.0, 0.0
	//}
	succ5 := int64(rand.Intn(succ8-0+1) + 0)
	if succ5 < config.Succ {
		a := "[aim]失败," + strconv.FormatInt(succ5, 10) + "%<" + strconv.FormatInt(config.Succ, 10) + "%\\n使用" + strconv.FormatFloat(up1, 'f', -1, 64) + "," + strconv.FormatFloat(usePower, 'f', -1, 64) + ""
		return "fail", "fail", a, up1, usePower
	} else {
		succ3 := succ5 - config.Succ
		succ4 := strconv.FormatInt(succ3, 10)
		succ6, _ := strconv.ParseFloat(succ4, 64)
		//if err != nil {
		//	over = true
		//	return "", "", "", 0.0, 0.0
		//}
		//succ1 = succ2 - config.Succ
		succ := succ6 / succ1
		maxbs := up*config.AddBuildSpeed + config.MaxBuildSpeed - config.MinBuildSpeed*succ
		bs := maxbs + config.MinBuildSpeed
		maxms := up*config.AddMoveSpeed + config.MaxMoveSpeed - config.MinMoveSpeed*succ
		ms := maxms + config.MinMoveSpeed
		maxmis := up*config.AddMineSpeed + config.MaxMineSpeed - config.MinMineSpeed*succ
		mis := maxmis + config.MinMineSpeed
		maxhp := up*config.AddHealth + config.MaxHealth - config.MinHealth*succ
		hp := maxhp + config.MinHealth
		maxamo := up*config.AddAmount + config.MaxAmount - config.MinAmount*succ
		amo := maxamo + config.MinAmount
		maxam := up*config.AddAmmoCaption + config.MaxAmmoCaption - config.MinAmmoCaption*succ
		am := maxam + config.MinAmmoCaption
		maxit := up*config.AddItemCaption + config.MaxItemCaption - config.MinItemCaption*succ
		it := maxit + config.MinItemCaption
		//e := 'f'
		out := ";u.buildSpeed=" + strconv.FormatFloat(bs, 'f', -1, 64) + ";u.moveSpeed=" + strconv.FormatFloat(ms, 'f', -1, 64) + ";u.mineSpeed=" + strconv.FormatFloat(mis, 'f', -1, 64) + ";u.maxHealth=" + strconv.FormatFloat(hp, 'f', -1, 64) + ";u.maxAmmoCaption" + strconv.FormatFloat(am, 'f', -1, 64) + ";u.itemCaption=" + strconv.FormatFloat(it, 'f', -1, 64)
		amo1 := strconv.FormatFloat(amo, 'f', -1, 64)
		aa := strconv.FormatInt(config.Succ, 10)
		ab := strconv.FormatFloat(up1, 'f', -1, 64)
		ac := strconv.FormatFloat(usePower, 'f', -1, 64)
		a := "[aim]成功," + strconv.FormatInt(succ5, 10) + "%>" + aa + "%\\n使用" + ab + "," + ac + ""
		return amo1, out, a, up1, usePower
	}
}
func (this *AimManager) admin(command []string, uuid string, userName string) bool {
	if this.mdt.users[uuid].IsAdmin == false {
		this.jsSay("[aim][red]你没有权限做这件事!")
		over = true
		return false
	}
	this.jsSay(this.mdt.users[uuid].Name)
	if len(command) < 3 {
		this.jsSay("[aim][red]命令格式错误!输入!aim help查看帮助。")
		over = true
		return false
	}
	cmdName := command[2]
	if strings.HasPrefix(cmdName, "sp") {
		this.spawn(command)
	} else if strings.HasPrefix(cmdName, "s") {
		this.setblock(command)
	} else if strings.HasPrefix(cmdName, "f") {
		this.fill(command)
	} else if strings.HasPrefix(cmdName, "k") {
		this.kill(command)
	} else if strings.HasPrefix(cmdName, "g") {
		this.gamerule(command)
	} else if strings.HasPrefix(cmdName, "a") {
		this.aim(command)
	} else {
		this.jsSay("[aim][red]未知指令!输入!aim help查看帮助。")
	}
	over = true
	return true
}
func (this *AimManager) spawn(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) setblock(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) fill(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) kill(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) gamerule(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) aim(command []string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) help(command []string) bool {
	//this.jsSay("help")
	if len(command) < 3 {
		this.jsSay("[aim]--help1/1--\\nadmin <command>   管理\\nhelp [command/page] [command/page] 帮助\\ninfo 玩家信息\\nrun <command>     运行")
		over = true
		return true
	} else {
		cmdType, err := strconv.Atoi(command[2])
		if err == nil {
			if cmdType == 1 {
				this.jsSay("[aim]--help1/1--\\nadmin <command>   管理\\nhelp [command/page] [command/page] 帮助\\ninfo 玩家信息\\nrun <command>     运行")
				over = true
				return true
			} else if cmdType > 1 {
				this.jsSay("[aim][red]页码过大，只有1页。")
				over = true
				return true
			} else if cmdType < 1 {
				this.jsSay("[aim][red]页码过小，只有1页。")
				over = true
				return true
			}
		} else if strings.HasPrefix(command[2], "a") {
			if len(command) < 4 {
				this.jsSay("[aim]--help1/2--\\nspawn <team> <type> [amount] 生成单位\\nsetblock <x> <y> <team> <type> [rotate]设置方块\\nfill <x1> <y1> <x2> <y2> <size> <team> <type> [rotate] 填充方块")
				over = true
				return true
			} else {
				page, err := strconv.Atoi(command[3])
				if err != nil {
					over = true
					return false
				}
				if page == 1 {
					this.jsSay("[aim]--help1/2--\\nspawn <team> <type> [amount] 生成单位\\nsetblock <x> <y> <team> <type> [rotate]设置方块\\nfill <x1> <y1> <x2> <y2> <size> <team> <type> [rotate] 填充方块")
					over = true
					return true
				} else if page == 2 {
					this.jsSay("[aim]--help2/2--\\nkill [team] [type] 清除单位\\ngamerule <gamerule> <vaule> 设置游戏规则\\naim <set/add/rem> <player> <data> <vaule> 设置玩家的aim配置")
					over = true
					return true
				} else if page > 2 {
					this.jsSay("[aim][red]页码过大，只有2页。")
					over = true
					return true
				} else if page < 1 {
					this.jsSay("[aim][red]页码过小，只有2页。")
					over = true
					return true
				}
			}
		} else if strings.HasPrefix(command[2], "r") {
			if len(command) < 4 {
				this.jsSay("[aim]--help1/3--\\nmono [point] 召唤mono\\nmega [point] 召唤mega\\nflyboat [point] 召唤飞船\\nlancer [point] 召唤一群不能动的quasar")
				over = true
				return true
			} else {
				page, err := strconv.Atoi(command[3])
				if err != nil {
					over = true
					return false
				}
				if page == 1 {
					this.jsSay("[aim]--help1/3--\\nmono [point] 召唤mono\\nmega [point] 召唤mega\\nflyboat [point] 召唤飞船\\nlancer [point] 召唤一群不能动的quasar")
					over = true
					return true
				} else if page == 2 {
					this.jsSay("[aim]--help2/3--\\ncrepper [point] 召唤一群苦力怕")
					over = true
					return true
				} else if page == 3 {
					this.jsSay("[aim]--help3/3--\\ndrill <x> <y> [point] 可以钻破地形的钻头\\nshard 召唤一个小型核心\\nkillallunit 灭霸(清除所有单位)")
					over = true
					return true
				} else if cmdType > 3 {
					this.jsSay("[aim][red]页码过大，只有3页。")
					over = true
					return true
				} else if cmdType < 1 {
					this.jsSay("[aim][red]页码过小，只有3页。")
					over = true
					return true
				}
			}
		}
	}
	over = false
	return true
}
func (this *AimManager) info(uuid string) bool {
	data, has := this.aimInfo.AimUserInfo[uuid]
	data = data
	if !has {
		this.aimInfo.AimUserInfo[uuid] = make([]Aim, 0)
		for index, _ := range this.aimInfo.AimUserInfo[uuid] {
			aimm := &(this.aimInfo.AimUserInfo[uuid][index])
			aimm.MaxPoint = 35.0
			aimm.MaxPower = 35.0
			aimm.P_tTime = 120000
			aimm.Pw_tTime = 60000
			aimm.CanUseAim = true
		}
	}
	this.jsSay("1")
	for index, _ := range this.aimInfo.AimUserInfo[uuid] {
		info := &(this.aimInfo.AimUserInfo[uuid][index])
		this.jsSay("[aim]-用户信息-")
		//e := "f"
		this.jsSay(":" + strconv.FormatFloat(info.Point, 'f', -1, 64) + "/" + strconv.FormatFloat(info.MaxPoint, 'f', -1, 64))
		this.jsSay(":" + strconv.FormatFloat(info.Power, 'f', -1, 64) + "/" + strconv.FormatFloat(info.MaxPower, 'f', -1, 64))
		this.jsSay("[aim]-用户信息-")
	}
	over = true
	return true
}
func (this *AimManager) run(command []string, uuid string) bool {
	this.jsSay("run")
	x, y := this.getPlayerXY(this.mdt.users[uuid].Name)
	this.jsSay(x + "," + y)
	if len(command) < 3 {
		this.jsSay("[aim][red]命令格式错误!输入!aim help查看帮助。")
		over = true
		return true
	} else {
		cmdName := command[2]
		if strings.HasPrefix(cmdName, "mo") {
			this.mono(command, uuid)
			this.jsSay("mono")
		} else if strings.HasPrefix(cmdName, "m") {
			this.mega(command, uuid)
			this.jsSay("mega")
		} else if strings.HasPrefix(cmdName, "f") {
			this.flyboat(command, uuid)
		} else if strings.HasPrefix(cmdName, "l") {
			this.lancer(command, uuid)
		} else if strings.HasPrefix(cmdName, "c") {
			this.creeper(command, uuid)
		} else if strings.HasPrefix(cmdName, "d") {
			this.drill(command, uuid)
		} else if strings.HasPrefix(cmdName, "s") {
			this.shard(command, uuid)
		} else if strings.HasPrefix(cmdName, "k") {
			this.killallunit(command, uuid)
		} else {
			this.jsSay("[aim][red]未知指令!输入!aim help查看帮助。")
		}
	}
	over = true
	return true
}

func (this *AimManager) mono(command []string, uuid string) bool {
	var dat RandData
	{
		dat.MinPoint = 10.0
		dat.MaxPoint = 25.6
		dat.MinUsePower = 15
		dat.AddUsePower = 2
		dat.Succ = 20
		dat.AddSucc = 6
		dat.MinBuildSpeed = 0.0
		dat.MaxBuildSpeed = 0.0
		dat.AddBuildSpeed = 0.0
		dat.MinMoveSpeed = 1.5
		dat.MaxMoveSpeed = 1.7
		dat.AddMoveSpeed = 0.08
		dat.MinMineSpeed = 2.5
		dat.MaxMineSpeed = 2.8
		dat.AddMineSpeed = 0.11
		dat.MinItemCaption = 20.0
		dat.MaxItemCaption = 25.0
		dat.AddItemCaption = 1.0
		dat.MinAmmoCaption = 0.0
		dat.MaxAmmoCaption = 0.0
		dat.AddAmmoCaption = 0.0
		dat.MinHealth = 100.0
		dat.MaxHealth = 120.0
		dat.AddHealth = 5.0
		dat.MinAmount = 1.0
		dat.MaxAmount = 3.0
		dat.AddAmount = 0.5
	}

	timer := time.NewTimer(time.Duration(2) * time.Second)
	<-timer.C
	da, has := this.aimInfo.AimUserInfo[uuid]
	da = da
	if !has {
		over = true
		return false
	}
	for index, _ := range this.aimInfo.AimUserInfo[uuid] {
		info := &(this.aimInfo.AimUserInfo[uuid][index])
		a := ""
		if len(command) < 4 {
			a = "10.0"
		} else {
			a = command[3]
			a = a
		}
		amo, data, say, up, upw := this.randData(a, dat)
		if amo == "fail" {
			this.jsSay(say)
			over = true
			return false
		} else if info.CanUseAim == false {
			this.jsSay("[aim]power过载，请等待至恢复完成")
			over = true
			return false
		} else if info.Point < up {
			this.jsSay("[aim]point不足，需要" + strconv.FormatFloat(up, 'f', -1, 64))
			over = true
			return false
		} else {
			if info.Power < upw {
				info.CanUseAim = false
			}
			this.jsSay(say)
			info.Point = info.Point - up
			info.Power = info.Power - upw
			x, y := this.getPlayerXY(this.mdt.users[uuid].Name)
			this.runjs("for(i=" + amo + ";i>=1;i--){UnitTypes.mono.spawn(Team.sharded," + x + "," + y + ")" + data + "}")
			this.jsSay(uuid)
			over = true
		}
	}
	return true
}

func (this *AimManager) mega(command []string, uuid string) bool {
	var dat RandData
	{
		dat.MinPoint = 20.0
		dat.MaxPoint = 35.0
		dat.MinUsePower = 30
		dat.AddUsePower = 3
		dat.Succ = 35
		dat.AddSucc = 4
		dat.MinBuildSpeed = 2.6
		dat.MaxBuildSpeed = 2.8
		dat.AddBuildSpeed = 0.2
		dat.MinMoveSpeed = 3.0
		dat.MaxMoveSpeed = 3.5
		dat.AddMoveSpeed = 0.06
		dat.MinMineSpeed = 4.5
		dat.MaxMineSpeed = 5.8
		dat.AddMineSpeed = 0.09
		dat.MinItemCaption = 65.0
		dat.MaxItemCaption = 75.0
		dat.AddItemCaption = 2.0
		dat.MinAmmoCaption = 0.0
		dat.MaxAmmoCaption = 0.0
		dat.AddAmmoCaption = 0.0
		dat.MinHealth = 440.0
		dat.MaxHealth = 500.0
		dat.AddHealth = 7.0
		dat.MinAmount = 1.0
		dat.MaxAmount = 2.0
		dat.AddAmount = 0.25
	}

	timer := time.NewTimer(time.Duration(2) * time.Second)
	<-timer.C
	da, has := this.aimInfo.AimUserInfo[uuid]
	da = da
	if !has {
		over = true
		return false
	}
	for index, _ := range this.aimInfo.AimUserInfo[uuid] {
		info := &(this.aimInfo.AimUserInfo[uuid][index])
		a := "0.0"
		if len(command) < 4 {
			a = "30.0"
			a = a
		} else {
			a = command[3]
			a = a
		}
		amo, data, say, up, upw := this.randData(a, dat)
		if amo == "fail" {
			this.jsSay(say)
			over = true
			return false
		} else if info.CanUseAim == false {
			this.jsSay("[aim]power过载，请等待至恢复完成")
			over = true
			return false
		} else if info.Point < up {
			this.jsSay("[aim]point不足，需要" + strconv.FormatFloat(up, 'f', -1, 64))
			over = true
			return false
		} else {
			if info.Power < upw {
				info.CanUseAim = false
			}
			this.jsSay(say)
			info.Point = info.Point - up
			info.Power = info.Power - upw
			x, y := this.getPlayerXY(this.mdt.users[uuid].Name)
			this.runjs("for(i=" + amo + ";i>=1;i--){UnitTypes.mega.spawn(Team.sharded," + x + "," + y + ")" + data + "}")
			this.jsSay(uuid)
			over = true
		}
	}
	return true
}
func (this *AimManager) flyboat(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")

	over = true
	return true
}
func (this *AimManager) lancer(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) creeper(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) drill(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) shard(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}
func (this *AimManager) killallunit(command []string, uuid string) bool {
	this.jsSay("[aim][red]指令未完成!")
	over = true
	return true
}

func (this *AimManager) getXY(name string) (string, string) {
	//timer := time.NewTimer(time.Duration(2) * time.Second)
	//<-timer.C
	x := "0"
	y := "0"
	xy := strings.Split(jsdata, ";")
	for i, dt := range xy {
		xxyy := strings.Split(dt, ",")
		i = i
		if len(xxyy) < 1 {
			x = "0"
			y = "0"
		}
		if xxyy[0] == name {
			x = string(xxyy[1])
			y = string(xxyy[2])
		}
	}
	return x, y

}

func (this *AimManager) getPlayerXY(name string) (string, string) {
	this.runjs("var data=\"[jsdata]\"")
	this.runjs("function xy(p){if (p.player!=null){data=data+p.player.name+\",\"+p.x+\",\"+p.y+\";\"}}")
	this.runjs("data=\"[jsdata]\";Groups.unit.each(u=>xy(u));data")
	x, y := this.getXY(name)
	return x, y
}

func (this *Mindustry) proc_aim(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	if over == false {
		this.aim.jsSay("[aim]其他指令正在执行！")
		return false
	}
	over = false
	//aim不需要判断是否在投票
	this.aim.jsSay(uuid)
	this.aim.jsSay(userInput)
	command := strings.Split(userInput, " ")
	if len(command) < 2 {
		this.aim.jsSay("[aim][red]命令格式错误!输入!aim help查看帮助。")
		over = true
		return false
	}
	cmdName := command[1]
	fmt.Printf(cmdName)
	this.aim.aimInfoTick(uuid)
	if strings.HasPrefix(cmdName, "a") {
		this.aim.admin(command, uuid, userName)
	} else if strings.HasPrefix(cmdName, "h") {
		this.aim.help(command)
	} else if strings.HasPrefix(cmdName, "i") {
		this.aim.info(uuid)
	} else if strings.HasPrefix(cmdName, "r") {
		this.aim.run(command, uuid)
	} else {
		this.aim.jsSay("[aim][red]未知指令!输入!aim help查看帮助。")
	}
	over = true
	return true
}
