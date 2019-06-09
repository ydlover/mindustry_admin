package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kortemy/lingo"
	"github.com/larspensjo/config"
	"github.com/robfig/cron"
)

var _VERSION_ = "1.0"

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripColor(str string) string {
	return re.ReplaceAllString(str, "")
}

type UserCmdProcHandle func(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool

type Admin struct {
	Name         string `json:"name"`
	Id           string `json:"id"`
	LastVistTime string `json:"last vist time"`
}
type AdminCfg struct {
	SuperAdminList []Admin `json:"super admin"`
	AdminList      []Admin `json:"admin"`
}

type Ban struct {
	Name   string `json:"name"`
	Id     string `json:"id"`
	Ip     string `json:"ip"`
	Reason string `json:"reason"`
	Time   string `json:"time"`
}
type BanCfg struct {
	BanList []Ban `json:"ban list"`
}
type User struct {
	name         string
	isAdmin      bool
	isSuperAdmin bool
	level        int
}
type Cmd struct {
	name   string
	level  int
	isVote bool
}

type Mindustry struct {
	name                   string
	adminCfg               *AdminCfg
	jarPath                string
	users                  map[string]User
	votetickUsers          map[string]int
	serverOutR             *regexp.Regexp
	cfgAdminCmds           string
	cfgSuperAdminCmds      string
	cfgNormCmds            string
	cfgVoteCmds            string
	cmds                   map[string]Cmd
	cmdHelps               map[string]string
	port                   int
	mode                   string
	cmdFailReason          string
	currProcCmd            string
	notice                 string //cron task auto notice msg
	playCnt                int
	serverIsStart          bool
	serverIsRun            bool
	maps                   []string
	userCmdProcHandles     map[string]UserCmdProcHandle
	l                      *lingo.L
	i18n                   lingo.T
	banCfg                 string
	remoteBanCfg           *BanCfg
	m_isPermitMapModify    bool
	isShowDefaultMapInMaps bool
}

func (this *Mindustry) getAdminList(adminList []Admin, isShowWarn bool) string {
	list := ""
	for _, admin := range adminList {
		if list != "" {
			list += ","
		}
		if isShowWarn {
			if admin.Id == "" {
				list += "[yellow]"
			} else {
				list += "[green]"
			}
		}
		list += admin.Name
		if admin.LastVistTime != "" {
			list += "("
			list += admin.LastVistTime
			list += ")"
		}
	}
	return list

}
func findInBan(banCfgList []Ban, id string) bool {
	for _, currBan := range banCfgList {
		if id == currBan.Id {
			return true
		}
	}
	return false
}
func (this *Mindustry) netBan(in io.WriteCloser) {
	if this.banCfg == "" {
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(5 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*5)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
		},
	}
	request, netErr := http.NewRequest("GET", this.banCfg, nil)
	if netErr != nil {
		log.Printf("[ERR]Load remote banb cfg fail:%s,netError:%v!\n", this.banCfg, netErr)
		return
	}
	response, responseErr := client.Do(request)
	if responseErr != nil {
		log.Printf("[ERR]Load remote banb cfg fail:%s,netError:%v!\n", this.banCfg, responseErr)
		return
	}
	if response.StatusCode == 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("[ERR]Load remote banb cfg io fail:%s!\n", this.banCfg)
			return
		}
		var currRemoteBanCfg = BanCfg{}
		err = json.Unmarshal(body, &currRemoteBanCfg)
		if err != nil {
			log.Printf("[ERR]Load remote banb cfg fail:%s!\n", this.banCfg)
			return
		}
		isSame := true
		if len(this.remoteBanCfg.BanList) == len(currRemoteBanCfg.BanList) {
			for _, remoteBan := range this.remoteBanCfg.BanList {
				if !findInBan(currRemoteBanCfg.BanList, remoteBan.Id) {
					isSame = false
					break
				}
			}
		} else {
			isSame = false
		}
		if isSame {
			return
		}
		var unbanList []string
		var banList []string
		unbanList = make([]string, 0)
		banList = make([]string, 0)
		for _, remoteBan := range this.remoteBanCfg.BanList {
			if !findInBan(currRemoteBanCfg.BanList, remoteBan.Id) {
				unbanList = append(unbanList, remoteBan.Id)
			}
		}

		for _, currBan := range currRemoteBanCfg.BanList {
			if !findInBan(this.remoteBanCfg.BanList, currBan.Id) {
				banList = append(banList, currBan.Id)
			}
		}
		for _, id := range unbanList {
			this.execCmd(in, "unban "+id)
		}
		for _, id := range banList {
			this.execCmd(in, "ban id "+id)
		}
		*this.remoteBanCfg = currRemoteBanCfg
	} else {
		log.Printf("[ERR]Load remote banb cfg fail:%s,remote response:%d!\n", this.banCfg, response.StatusCode)
	}

}
func (this *Mindustry) loadAdminConfig() {
	data, err := ioutil.ReadFile("admin.json")
	if err != nil {
		log.Printf("[ERR]Not found admin.json!\n")
		return
	}
	err = json.Unmarshal(data, this.adminCfg)
	if err != nil {
		log.Printf("[ERR]Load cfg fail:admin.json!\n")
		return
	}
	for _, admin := range this.adminCfg.SuperAdminList {
		log.Printf("SuperAdmin:%s(%s)\n", admin.Name, admin.Id)
	}
	for _, admin := range this.adminCfg.AdminList {
		log.Printf("Admin:%s(%s)\n", admin.Name, admin.Id)
	}
}
func WriteConfig(cfg string, jsonByte []byte) {
	f, err := os.OpenFile(cfg, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Printf("write config file %s fail:%v\n", cfg, err)
	}
	defer f.Close()
	f.Write(jsonByte)
	f.Sync()
}
func (this *Mindustry) writeAdminConfig() {
	data, err := json.MarshalIndent(this.adminCfg, "", "    ")
	if err != nil {
		log.Println("[ERR]writeAdminCfg fail:", err)
		return
	}
	WriteConfig("admin.json", data)
}
func (this *Mindustry) loadConfig() {
	this.l = lingo.New("en_US", "./locale")
	cfg, err := config.ReadDefault("config.ini")
	if err != nil {
		log.Println("[ini]not find config.ini,use default config")
		return
	}
	if cfg.HasSection("server") {
		_, err := cfg.SectionOptions("server")
		if err == nil {
			optionValue := ""
			optionValue, err = cfg.String("server", "superAdminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				this.cfgSuperAdminCmds = optionValue
				log.Printf("[ini]found superAdminCmds:%v\n", cmds)
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 9, false}
				}
			}

			optionValue, err = cfg.String("server", "adminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found adminCmds:%v\n", cmds)
				this.cfgAdminCmds = optionValue
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 1, false}
				}
			}
			optionValue, err = cfg.String("server", "normCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found normCmds:%v\n", cmds)
				this.cfgNormCmds = optionValue
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 0, false}
				}
			}
			optionValue, err = cfg.String("server", "votetickCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found votetickCmds:%v\n", cmds)
				this.cfgVoteCmds = optionValue
				for _, cmd := range cmds {
					if c, ok := this.cmds[cmd]; ok {
						c.isVote = true
						this.cmds[cmd] = c
					} else {
						log.Printf("[ini]votetick not found cmd:%s\n", cmd)
					}
				}
			}

			optionValue, err = cfg.String("server", "name")
			if err == nil {
				name := strings.TrimSpace(optionValue)
				this.name = name
			}
			optionValue, err = cfg.String("server", "jarPath")
			if err == nil {
				jarPath := strings.TrimSpace(optionValue)
				this.jarPath = jarPath
			}
			optionValue, err = cfg.String("server", "notice")
			if err == nil {
				notice := strings.TrimSpace(optionValue)
				this.notice = notice
			}

			optionValue, err = cfg.String("server", "language")
			if err == nil {
				languageCfg := strings.TrimSpace(optionValue)
				if languageCfg != "" {
					log.Printf("[ini]lanage cfg:%s\n", languageCfg)
					this.i18n = this.l.TranslationsForLocale(languageCfg)
				} else {
					this.i18n = this.l.TranslationsForLocale("en_US")
					log.Printf("[ini]lanage cfg invalid,use english\n")
				}
			} else {
				this.i18n = this.l.TranslationsForLocale("en_US")
				log.Printf("[ini]lanage cfg invalid,use english\n")
			}

			optionValue, err = cfg.String("server", "banCfg")
			if err == nil {
				banCfg := strings.TrimSpace(optionValue)
				this.banCfg = banCfg
			}
			optionValue, err = cfg.String("server", "isShowDefaultMapInMaps")
			if err == nil {
				this.isShowDefaultMapInMaps = strings.TrimSpace(optionValue) == "1"
			}

		}
	}

}
func (this *Mindustry) init() {
	this.serverOutR, _ = regexp.Compile(".*(\\[INFO\\]|\\[ERR\\])(.*)")
	this.users = make(map[string]User)
	this.votetickUsers = make(map[string]int)
	this.cmds = make(map[string]Cmd)
	this.cmdHelps = make(map[string]string)
	this.userCmdProcHandles = make(map[string]UserCmdProcHandle)
	rand.Seed(time.Now().UnixNano())
	this.name = fmt.Sprintf("mindustry-%d", rand.Int())
	this.jarPath = "server-release.jar"
	this.serverIsStart = true
	this.adminCfg = new(AdminCfg)
	this.remoteBanCfg = new(BanCfg)
	this.loadConfig()
	this.loadAdminConfig()
	this.userCmdProcHandles["admin"] = this.proc_admin
	this.userCmdProcHandles["unadmin"] = this.proc_unadmin
	this.userCmdProcHandles["directCmd"] = this.proc_directCmd
	this.userCmdProcHandles["gameover"] = this.proc_gameover
	this.userCmdProcHandles["help"] = this.proc_help
	this.userCmdProcHandles["host"] = this.proc_host
	this.userCmdProcHandles["hostx"] = this.proc_host
	this.userCmdProcHandles["save"] = this.proc_save
	this.userCmdProcHandles["load"] = this.proc_load
	this.userCmdProcHandles["maps"] = this.proc_maps
	this.userCmdProcHandles["status"] = this.proc_status
	this.userCmdProcHandles["slots"] = this.proc_slots
	this.userCmdProcHandles["admins"] = this.proc_admins
	this.userCmdProcHandles["show"] = this.proc_show
	this.userCmdProcHandles["votetick"] = this.proc_votetick
	this.userCmdProcHandles["mode"] = this.proc_mode
	this.userCmdProcHandles["mapManage"] = this.proc_mapManage

}

var colorCodeReg = regexp.MustCompile(`\[.*?\]|#[0-9a-fA-F]+`)

func removeColorCode(str string) string {
	newStr := colorCodeReg.ReplaceAllString(str, "")
	return newStr
}

func (this *Mindustry) execCommand(commandName string, params []string) error {
	cmd := exec.Command(commandName, params...)
	fmt.Println(cmd.Args)
	stdout, outErr := cmd.StdoutPipe()
	stdin, inErr := cmd.StdinPipe()
	if outErr != nil {
		return outErr
	}

	if inErr != nil {
		return inErr
	}
	cmd.Start()
	go func(cmd *exec.Cmd) {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		s := <-c
		if cmd.Process != nil {
			log.Printf("sub process exit:%s", s)
			cmd.Process.Kill()
		}
	}(cmd)
	c := cron.New()
	spec := "0 0 * * * ?"
	c.AddFunc(spec, func() {
		this.hourTask(stdin)
	})
	spec = "0 5/10 * * * ?"
	c.AddFunc(spec, func() {
		this.tenMinTask(stdin)
	})
	c.Start()
	go func(cmd *exec.Cmd) {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			inputCmd := strings.TrimRight(line, "\n")
			if inputCmd == "stop" || inputCmd == "exit" {
				this.serverIsStart = false
				this.serverIsRun = false
				this.writeAdminConfig()
			}
			if inputCmd == "host" || inputCmd == "load" {
				this.serverIsStart = true
			}
			this.execCmd(stdin, inputCmd)
		}
	}(cmd)

	reader := bufio.NewReader(stdout)

	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Printf(line)
		this.output(StripColor(line), stdin)
	}
	cmd.Wait()
	return nil
}
func (this *Mindustry) hourTask(in io.WriteCloser) {
	hour := time.Now().Hour()
	log.Printf("hourTask trig:%d\n", hour)
	if this.serverIsRun {
		this.execCmd(in, "save "+strconv.Itoa(hour))
		this.say(in, "info.auto_save", hour)
		this.netBan(in)
	} else {
		log.Printf("game is not running.\n")
	}
}

func (this *Mindustry) tenMinTask(in io.WriteCloser) {
	log.Printf("tenMinTask trig[%.3f°C].\n", getCpuTemp())

	if !this.serverIsStart {
		return
	}
	if !this.serverIsRun {
		log.Printf("game is not running,exit.\n")
		this.execCmd(in, "exit")
	} else {
		this.say(in, this.notice)
		log.Printf("update game status.\n")
		this.currProcCmd = "status"
		this.execCmd(in, "status ")
	}
}
func (this *Mindustry) addUser(name string) {
	if _, ok := this.users[name]; ok {
		return
	}
	this.users[name] = User{name, false, false, 0}
	log.Printf("add user info :%s\n", name)
}
func (this *Mindustry) onlineAdmin(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("user %s not found\n", name)
		return
	}
	tempUser := this.users[name]
	tempUser.isAdmin = true
	if tempUser.level < 1 {
		tempUser.level = 1
	}
	this.users[name] = tempUser
	log.Printf("online admin :%s\n", name)
}

func (this *Mindustry) onlineSuperAdmin(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("user %s not found\n", name)
		return
	}
	tempUser := this.users[name]
	tempUser.isAdmin = true
	tempUser.isSuperAdmin = true
	if tempUser.level < 9 {
		tempUser.level = 9
	}
	this.users[name] = tempUser
	log.Printf("online superAdmin :%s\n", name)
}

func getNowDate() string {
	return time.Now().Format("06-01-02")
}
func (this *Mindustry) judgeAndUpdateAdmin(admin *Admin, name string, uuid string) (bool, bool) {
	if admin.Name != name {
		return false, false
	}
	if admin.Id == "" {
		admin.Id = uuid
		admin.LastVistTime = getNowDate()
		log.Printf("admin %s[%s] first login.\n", name, uuid)
		return true, true
	}
	if admin.Id != uuid {
		return false, false
	}
	nowDate := getNowDate()
	if nowDate == admin.LastVistTime {
		return true, false
	}
	admin.LastVistTime = nowDate
	return true, true
}

const (
	NORM = iota
	ADMIN
	SUPER_ADMIN
)

func (this *Mindustry) judgeRole(role int, adminList []Admin, name string, uuid string) (int, bool) {
	isAdmin := false
	isUpdate := false
	for index, admin := range adminList {
		isTempAdmin, isTempUpdate := this.judgeAndUpdateAdmin(&admin, name, uuid)
		if !isTempAdmin {
			continue
		}
		isAdmin = true
		if isTempUpdate {
			isUpdate = true
			adminList[index] = admin
		}
		break
	}
	if isAdmin {
		return role, isUpdate
	} else {
		return NORM, false
	}
}

func (this *Mindustry) getUserRole(name string, uuid string) int {
	role, isUpdate := this.judgeRole(ADMIN, this.adminCfg.AdminList, name, uuid)
	if role == NORM {
		role, isUpdate = this.judgeRole(SUPER_ADMIN, this.adminCfg.SuperAdminList, name, uuid)
	}
	if isUpdate {
		this.writeAdminConfig()
	}
	return role
}

func (this *Mindustry) onlineUser(name string, uuid string) {
	this.playCnt++

	if _, ok := this.users[name]; ok {
		return
	}
	this.addUser(name)
	role := this.getUserRole(name, uuid)
	if role == SUPER_ADMIN {
		this.onlineSuperAdmin(name)
	} else if role == ADMIN {
		this.onlineAdmin(name)
	}
}
func (this *Mindustry) offlineUser(name string, uuid string) {
	if this.playCnt > 0 {
		this.playCnt--
	}

	if _, ok := this.users[name]; !ok {
		return
	}

	this.delUser(name)
	return
}
func (this *Mindustry) delUser(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("del user not exist :%s\n", name)
		return
	}
	delete(this.users, name)
	log.Printf("del user info :%s\n", name)
}
func (this *Mindustry) execCmd(in io.WriteCloser, cmd string) {
	if cmd == "stop" || cmd == "host" || cmd == "hostx" || cmd == "load" {
		this.playCnt = 0
	}
	log.Printf("execCmd :%s\n", cmd)
	data := []byte(cmd + "\n")
	in.Write(data)
}
func (this *Mindustry) say(in io.WriteCloser, strKey string, v ...interface{}) {
	localeStr := "say " + this.i18n.Value(strKey) + "\n"
	info := fmt.Sprintf(localeStr, v...)
	in.Write([]byte(info))
}

func checkSlotValid(slot string) bool {
	files, _ := ioutil.ReadDir("./config/saves")
	for _, f := range files {
		if f.Name() == slot+".msav" {
			return true
		}
	}
	return false
}
func getSlotList() string {
	slotList := []string{}
	files, _ := ioutil.ReadDir("./config/saves")
	for _, f := range files {
		if strings.Count(f.Name(), "backup") > 0 {
			continue
		}
		if strings.HasSuffix(f.Name(), ".msav") {
			slotList = append(slotList, f.Name()[:len(f.Name())-len(".msav")])
		}
	}
	return strings.Join(slotList, ",")
}
func (this *Mindustry) cmdWaitTimeout(in io.WriteCloser, userName string, userInput string, cmdName string) {
	go func() {
		timer := time.NewTimer(time.Duration(5) * time.Second)
		<-timer.C
		if this.currProcCmd != "" {
			this.say(in, "error.cmd_timeout", this.currProcCmd)
			this.currProcCmd = ""
		}
	}()
	this.currProcCmd = cmdName
}

func (this *Mindustry) proc_maps(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.cmdWaitTimeout(in, userName, userInput, "maps")
	this.execCmd(in, "reloadmaps")
	this.maps = this.maps[0:0]
	this.execCmd(in, "maps")
	return true
}
func (this *Mindustry) proc_status(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.cmdWaitTimeout(in, userName, userInput, "status")
	this.execCmd(in, "status")

	return true
}
func (this *Mindustry) proc_host(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	mapName := ""
	temps := strings.Split(userInput, " ")
	if len(temps) < 2 {
		this.say(in, "error.cmd_length_invalid", userInput)
		return false
	}
	inputCmd := strings.TrimSpace(temps[0])
	inputMap := strings.TrimSpace(temps[1])
	inputMode := ""
	if this.mode != "" {
		if len(temps) > 2 {
			this.say(in, "error.cmd_host_fix_mode", this.mode)
			return false
		}
		inputMode = this.mode
	}
	if len(temps) > 2 {
		inputMode = strings.TrimSpace(temps[2])
	}
	if inputCmd == "hostx" {
		inputIndex := 0
		var err error = nil
		if inputIndex, err = strconv.Atoi(inputMap); err != nil {
			this.say(in, "error.cmd_hostx_id_not_number", userInput)
			return false
		}
		if inputIndex < 0 || inputIndex >= len(this.maps) {

			this.say(in, "error.cmd_hostx_id_not_found", userInput)
			return false
		}
		mapName = this.maps[inputIndex]
	} else if inputCmd == "host" {
		isFind := false
		for _, name := range this.maps {
			if name == inputMap {
				isFind = true
				mapName = name
				break
			}
		}
		if !isFind {
			this.say(in, "error.cmd_host_map_not_found", userInput)
			return false
		}
	} else {
		this.say(in, "error.cmd_invalid", userInput)
		return false
	}
	if !checkMode(inputMode) {
		this.say(in, "error.cmd_host_mode_invalid", userInput)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.say(in, "info.server_restart")
	this.execCmd(in, "reloadmaps")
	time.Sleep(time.Duration(5) * time.Second)
	this.execCmd(in, "stop")
	time.Sleep(time.Duration(5) * time.Second)
	mapName = strings.Replace(mapName, " ", "_", -1)
	if inputMode == "" {
		this.execCmd(in, "host "+mapName)
	} else {
		this.execCmd(in, "host "+mapName+" "+inputMode)
	}
	return true
}

func (this *Mindustry) proc_save(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	targetSlot := ""
	if userInput == "save" {
		minute := time.Now().Minute()
		targetSlot = fmt.Sprintf("%d%02d%02d", time.Now().Day(), time.Now().Hour(), minute/10*10)
	} else {
		targetSlot = userInput[len("save"):]
		targetSlot = strings.TrimSpace(targetSlot)
	}
	if _, ok := strconv.Atoi(targetSlot); ok != nil {
		this.say(in, "error.cmd_save_slot_invalid", targetSlot)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.execCmd(in, "save "+targetSlot)
	this.say(in, "info.save_slot_succ", targetSlot)
	return true
}

func (this *Mindustry) proc_load(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	targetSlot := userInput[len("load"):]
	targetSlot = strings.TrimSpace(targetSlot)
	if !checkSlotValid(targetSlot) {
		this.say(in, "error.cmd_load_slot_invalid", targetSlot)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.say(in, "info.server_restart")
	time.Sleep(time.Duration(5) * time.Second)
	this.execCmd(in, "stop")
	time.Sleep(time.Duration(5) * time.Second)
	this.execCmd(in, userInput)
	return true
}
func (this *Mindustry) proc_admin(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	targetName := userInput[len("admin"):]
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		this.say(in, "error.cmd_admin_name_invalid")
		return false
	}
	if isOnlyCheck {
		return true
	}
	newAdmin := Admin{targetName, "", ""}
	this.adminCfg.AdminList = append(this.adminCfg.AdminList, newAdmin)
	this.writeAdminConfig()
	this.say(in, "info.admin_added", targetName)

	return true
}
func (this *Mindustry) proc_unadmin(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	targetName := userInput[len("unadmin"):]
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		this.say(in, "error.cmd_admin_name_invalid")
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.execCmd(in, "unadmin "+targetName)
	for i, admin := range this.adminCfg.AdminList {
		if admin.Name == targetName {
			this.adminCfg.AdminList = append(this.adminCfg.AdminList[:i], this.adminCfg.AdminList[i+1:]...)
			this.writeAdminConfig()
			this.say(in, "info.admin_removed", targetName)
			return true
		}
	}
	this.say(in, "error.admin_unremoved", targetName)
	return false
}

func (this *Mindustry) proc_directCmd(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.execCmd(in, userInput)
	return true
}
func (this *Mindustry) proc_gameover(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.execCmd(in, "reloadmaps")
	this.execCmd(in, userInput)
	return true
}
func (this *Mindustry) proc_help(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	temps := strings.Split(userInput, " ")
	if len(temps) >= 2 {
		cmd := strings.TrimSpace(temps[1])
		this.say(in, "helps."+cmd, cmd)
	} else {
		if this.users[userName].isSuperAdmin {
			this.say(in, "info.super_admin_cmd", this.cfgSuperAdminCmds)
		} else if this.users[userName].isAdmin {
			this.say(in, "info.admin_cmd", this.cfgAdminCmds)
		} else {
			this.say(in, "info.user_cmd", this.cfgNormCmds)
		}
		this.say(in, "info.votetick_cmd", this.cfgVoteCmds)

	}
	return true
}

var tempOsPath = "/sys/class/thermal/thermal_zone0/temp"

func getCpuTemp() float64 {
	raw, err := ioutil.ReadFile(tempOsPath)
	if err != nil {
		log.Printf("Failed to read temperature from %q: %v", tempOsPath, err)
		return 0.0
	}

	cpuTempStr := strings.TrimSpace(string(raw))
	cpuTempInt, err := strconv.Atoi(cpuTempStr) // e.g. 55306
	if err != nil {
		log.Printf("%q does not contain an integer: %v", tempOsPath, err)
		return 0.0
	}
	cpuTemp := float64(cpuTempInt) / 1000.0
	//debug.Printf("CPU temperature: %.3f°C", cpuTemp)
	return cpuTemp
}
func (this *Mindustry) proc_show(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.say(in, "info.ver", _VERSION_)
	this.say(in, "info.cpu_temperature", getCpuTemp())
	return true
}
func (this *Mindustry) proc_admins(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	isShowWarn := this.users[userName].isSuperAdmin
	this.say(in, "info.super_admin_list", this.getAdminList(this.adminCfg.SuperAdminList, isShowWarn))
	this.say(in, "info.admin_list", this.getAdminList(this.adminCfg.AdminList, isShowWarn))
	return true

}

func (this *Mindustry) proc_slots(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.say(in, "info.slots_list", getSlotList())
	return true
}
func (this *Mindustry) checkVote() (bool, int, int) {
	if this.playCnt == 0 {
		log.Printf("playCnt is zero!\n")
		return false, 0, 0
	}
	agreeCnt := 0
	adminAgainstCnt := 0
	for userName, isAgree := range this.votetickUsers {
		if isAgree == 1 {
			agreeCnt++
		} else if _, ok := this.users[userName]; ok {
			if this.users[userName].isAdmin {
				adminAgainstCnt++
			}
		}
	}
	if adminAgainstCnt > 0 {
		return false, agreeCnt, adminAgainstCnt
	}

	return float32(agreeCnt)/float32(this.playCnt) >= 0.5, agreeCnt, adminAgainstCnt
}
func (this *Mindustry) proc_votetick(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	index := strings.Index(userInput, " ")
	if index < 0 {
		this.say(in, "error.cmd_votetick_target_invalid", userInput)
		return false
	}

	if len(this.votetickUsers) > 0 {
		this.say(in, "error.cmd_votetick_in_progress")
		return false
	}
	votetickCmd := strings.TrimSpace(userInput[index:])
	votetickCmdHead := votetickCmd
	index = strings.Index(votetickCmd, " ")
	if index >= 0 {
		votetickCmdHead = strings.TrimSpace(votetickCmd[:index])
	}

	if cmd, ok := this.cmds[votetickCmdHead]; ok {
		if !cmd.isVote {
			this.say(in, "error.cmd_votetick_not_permit", votetickCmdHead)
			return false
		}
	} else {
		this.say(in, "error.cmd_votetick_cmd_error", votetickCmdHead)
		return false
	}
	if handleFunc, ok := this.userCmdProcHandles[votetickCmdHead]; ok {
		checkRslt := handleFunc(in, userName, votetickCmd, true)
		if !checkRslt {
			return false
		}

		if isOnlyCheck {
			return true
		}

		this.currProcCmd = "votetick"
		this.votetickUsers = make(map[string]int)
		this.votetickUsers[userName] = 1
		go func() {
			timer := time.NewTimer(time.Duration(60) * time.Second)
			<-timer.C
			isSucc, agreeCnt, adminAgainstCnt := this.checkVote()
			if isSucc {
				this.say(in, "info.votetick_pass", this.playCnt, agreeCnt)
				handleFunc(in, userName, votetickCmd, false)
			} else {
				this.say(in, "info.votetick_fail", this.playCnt, agreeCnt, adminAgainstCnt)
			}
			this.votetickUsers = make(map[string]int)
			this.currProcCmd = ""
		}()

	} else {
		this.say(in, "error.cmd_votetick_cmd_not_support", votetickCmd)
		return false
	}
	this.say(in, "info.votetick_begin_info")
	return true
}
func checkMode(inputMode string) bool {
	if inputMode != "pvp" && inputMode != "attack" && inputMode != "" && inputMode != "sandbox" && inputMode != "survival" {
		return false
	}
	return true
}
func (this *Mindustry) proc_mode(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	temps := strings.Split(userInput, " ")
	if len(temps) < 2 {
		this.say(in, "info.mode_show", this.mode)
		return false
	}
	inputMode := temps[1]
	if inputMode == "none" {
		inputMode = ""
	}
	if !checkMode(inputMode) {
		this.say(in, "error.cmd_host_mode_invalid", userInput)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.mode = inputMode
	this.say(in, "info.mode_show", this.mode)
	return true
}

func (this *Mindustry) proc_mapManage(in io.WriteCloser, userName string, userInput string, isOnlyCheck bool) bool {
	if this.m_isPermitMapModify {
		this.say(in, "error.cmd_mapManage_started")
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.say(in, "info.cmd_mapManage_start")
	this.m_isPermitMapModify = true
	go func() {
		timer := time.NewTimer(time.Duration(600) * time.Second)
		<-timer.C
		this.m_isPermitMapModify = false
		this.say(in, "info.cmd_mapManage_end")
	}()
	return true
}
func (this *Mindustry) procUsrCmd(in io.WriteCloser, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	cmdName := temps[0]

	if cmd, ok := this.cmds[cmdName]; ok {
		if this.users[userName].level < cmd.level {
			this.say(in, "error.cmd_permission_denied", cmdName)
			return
		} else {
			if this.currProcCmd != "" {
				this.say(in, "error.cmd_is_exceuting", this.currProcCmd)
				return
			}

			if handleFunc, ok := this.userCmdProcHandles[cmdName]; ok {
				handleFunc(in, userName, userInput, false)
			} else {
				this.userCmdProcHandles["directCmd"](in, userName, userInput, false)
			}
		}

	} else {
		this.say(in, "error.cmd_invalid_user", cmdName)
	}
}
func (this *Mindustry) multiLineRsltCmdComplete(in io.WriteCloser, line string) bool {
	index := -1
	if this.currProcCmd == "maps" {
		if strings.Index(line, "Map directory:") >= 0 {
			mapsInfo := ""
			for index, name := range this.maps {
				if mapsInfo != "maps:" {
					mapsInfo += " "
				}
				mapsInfo += ("[cyan](" + strconv.Itoa(index) + ")[white]" + name)
			}
			this.say(in, "info.maps_list", mapsInfo)
			return true
		}
		mapNameEndIndex := -1
		index = strings.Index(line, ": Custom /")
		if index >= 0 {
			mapNameEndIndex = index
		}
		if this.isShowDefaultMapInMaps {
			index = strings.Index(line, ": Default /")
			if index >= 0 {
				mapNameEndIndex = index
			}
		}
		if mapNameEndIndex >= 0 {
			this.maps = append(this.maps, strings.TrimSpace(line[:mapNameEndIndex]))
		}
	} else if this.currProcCmd == "status" {

		index = strings.Index(line, "Players:")
		if index >= 0 {
			countStr := strings.TrimSpace(line[index+len("Players:")+1:])
			if count, ok := strconv.Atoi(countStr); ok == nil {
				this.playCnt = count
			}
			return true
		} else if strings.Index(line, "No players connected.") >= 0 {
			this.playCnt = 0
			return true
		} else if strings.Index(line, "Status: server closed") >= 0 {
			this.serverIsRun = false
			this.playCnt = 0

			return true
		}
	}
	return false
}

const USER_CONNECTED_KEY string = " has connected."
const USER_DISCONNECTED_KEY string = " has disconnected."
const SERVER_INFO_LOG string = "[INFO] "
const SERVER_ERR_LOG string = "[ERR!] "
const SERVER_READY_KEY string = "Server loaded. Type 'help' for help."
const SERVER_STSRT_KEY string = "Opened a server on port"

func getUserByOutput(key string, cmdBody string) (string, string, bool) {
	userInfo := strings.TrimSpace(cmdBody[:len(cmdBody)-len(key)])
	index := strings.Index(userInfo, "]")
	if index < 0 {
		return "", "", false
	}
	userName := strings.TrimSpace(userInfo[index+1:])
	uuid := strings.TrimSpace(userInfo[1:index])
	return userName, uuid, true
}
func (this *Mindustry) output(line string, in io.WriteCloser) {
	index := strings.Index(line, SERVER_ERR_LOG)
	if index >= 0 {
		errInfo := strings.TrimSpace(line[index+len(SERVER_ERR_LOG):])
		if strings.Contains(errInfo, "io.anuke.arc.util.ArcRuntimeException: File not found") {
			log.Printf("map not found , force exit!\n")
			this.execCmd(in, "exit")
		}
		this.cmdFailReason = errInfo
		return
	}

	index = strings.Index(line, SERVER_INFO_LOG)
	if index < 0 {
		return
	}
	cmdBody := strings.TrimSpace(line[index+len(SERVER_INFO_LOG):])
	if this.currProcCmd == "maps" || this.currProcCmd == "status" {
		//this.say(in, line)
		if this.multiLineRsltCmdComplete(in, cmdBody) {
			this.currProcCmd = ""
		}
		return
	}
	index = strings.Index(cmdBody, ":")
	if index > -1 {
		userName := strings.TrimSpace(cmdBody[:index])
		if _, ok := this.users[userName]; ok {
			if userName == "Server" {
				return
			}
			sayBody := strings.TrimSpace(cmdBody[index+1:])
			if strings.HasPrefix(sayBody, "\\") || strings.HasPrefix(sayBody, "/") || strings.HasPrefix(sayBody, "!") {
				this.procUsrCmd(in, userName, sayBody[1:])
			} else if len(this.votetickUsers) > 0 {
				if sayBody == "1" {
					log.Printf("%s votetick agree\n", userName)
					this.votetickUsers[userName] = 1
				} else if sayBody == "0" {
					log.Printf("%s votetick not agree\n", userName)
					this.votetickUsers[userName] = 0
				}
			} else {
				//fmt.Printf("%s : %s\n", userName, sayBody)
			}
		}
	}

	if strings.HasSuffix(cmdBody, USER_CONNECTED_KEY) {
		userName, uuid, isSucc := getUserByOutput(USER_CONNECTED_KEY, cmdBody)
		if !isSucc {
			log.Printf("[%s] invalid\n", cmdBody)
			return
		}
		if userName == "Server" {
			this.say(in, "error.login_forbbidden_username")
			this.execCmd(in, "kick "+userName)
			return
		}
		this.onlineUser(userName, uuid)

		if this.users[userName].isAdmin {
			time.Sleep(1 * time.Second)
			if this.users[userName].isSuperAdmin {
				this.say(in, "info.welcom_super_admin", userName)
			} else {
				this.say(in, "info.welcom_admin", userName)
			}
			this.execCmd(in, "admin "+userName)
		}

	} else if strings.HasSuffix(cmdBody, USER_DISCONNECTED_KEY) {
		userName, uuid, isSucc := getUserByOutput(USER_DISCONNECTED_KEY, cmdBody)
		if !isSucc {
			log.Printf("[%s] invalid\n", cmdBody)
			return
		}
		if userName == "Server" {
			this.say(in, "error.login_forbbidden_username")
			this.execCmd(in, "kick "+userName)
			return
		}
		this.offlineUser(userName, uuid)
	} else if strings.HasPrefix(cmdBody, SERVER_READY_KEY) {
		this.playCnt = 0
		this.serverIsRun = true
		this.netBan(in)
		this.netBan(in)
		this.netBan(in)

		this.execCmd(in, "name "+this.name)
		this.execCmd(in, "port "+strconv.Itoa(this.port))
		this.execCmd(in, "host Fortress")
	} else if strings.HasPrefix(cmdBody, SERVER_STSRT_KEY) {
		log.Printf("server starting!\n")
		this.serverIsRun = true
		this.playCnt = 0
	}
}
func (this *Mindustry) run() {
	var para = []string{"-jar", this.jarPath}
	for {
		this.execCommand("java", para)
		if this.serverIsStart {
			log.Printf("server crash,wait(10s) reboot!\n")
			time.Sleep(time.Duration(10) * time.Second)
		} else {
			break
		}
	}
}
func (this *Mindustry) isPermitMapModify() bool {
	return this.m_isPermitMapModify
}

func (this *Mindustry) startMapUpServer(port int) {
	go func(serverPort int) {
		StartFileUpServer(serverPort, this)
	}(port)
}
func main() {
	mode := flag.String("mode", "", "fix mode:survival,attack,sandbox,pvp")
	port := flag.Int("port", 6567, "Input port")
	map_port := flag.Int("up", 6569, "map up port")
	flag.Parse()
	log.Printf("version:%s!\n", _VERSION_)

	mindustry := Mindustry{}
	mindustry.init()
	mindustry.mode = *mode
	mindustry.port = *port
	mindustry.startMapUpServer(*map_port)
	mindustry.run()
}
