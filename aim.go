package main

import ()

type Aim struct {
	Uuid          string `json:"uuid"`
	Point         int    `json:"point"`
	MaxPoint      int    `json:"maxPoint"`
	Power         int    `json:"power"`
	MaxPower      int    `json:"maxPower"`
	FullPointTime int    `json:"fullPointTime"`
	FullPowerTime int    `json:"fullPowerTime"`
	CanUseAim     bool   `json:"canUseAim"`
}
type AimList struct {
	AimLst []Aim `json:"aim list"`
}
type AimInfo struct {
	AimUserInfo map[string]AimList `json:"aim_info"`
}
type AimManager struct {
	mdt     *Mindustry
	aimInfo *AimInfo
}

func (this *AimManager) Init(mdt *Mindustry) {
	this.mdt = mdt
	this.aimInfo = new(AimInfo)
	this.aimInfo.AimUserInfo = make(map[string]AimList, 0)
}

func (this *AimManager) runjs(js string) string {
	this.mdt.execCmd("js " + js)
	return ""
}

func (this *AimManager) jsSay(message string) string {
	this.runjs("Call.sendMessage(" + message + ")")
	return ""
}

// 如果要使用上面的函数使用this.aim.sayJs(xxx)
func (this *Mindustry) proc_aim(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.say(uuid)
	this.say(userInput)
	return true
}
