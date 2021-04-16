package main

import (
	"bufio"
	"container/list"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/kortemy/lingo"
	"github.com/larspensjo/config"
	"github.com/robfig/cron"
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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var _VERSION_ = "1.0"

func cryptoMd5(passwd string) string {
	h := md5.New()
	h.Write([]byte(passwd))
	return hex.EncodeToString(h.Sum(nil))
}

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripColor(str string) string {
	return re.ReplaceAllString(str, "")
}

type UserCmdProcHandle func(uuid string, userName string, userInput string, isOnlyCheck bool) bool

type Message struct {
	Id      int    `json:"id"`
	Message string `json:"message"`
	Time    string `json:"time"`
}
type HistoryMessages struct {
	Messages []Message `json:"mesaages"`
}

const MAX_MESSAGE_CACHE int = 100

type MessageManager struct {
	messageListMutex sync.RWMutex
	messageId        int
	max              int
	messages         list.List
	historyFileName  string
}

func (this *MessageManager) loadHistory(fileName string) {
	this.historyFileName = fileName
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Printf("[ERR]Not found message_board.json!\n")
		return
	}
	history := new(HistoryMessages)
	history.Messages = make([]Message, 0)
	err = json.Unmarshal(data, history)
	if err != nil {
		log.Printf("[ERR]Load history fail:%s,err:%v!\n", fileName, err)
		return
	}
	log.Printf("[INFO]Load history message board:%d!\n", len(history.Messages))
	for _, m := range history.Messages {
		this.messages.PushBack(m)
		this.messageId += 1
	}
}

func (this *MessageManager) init(max int) {
	this.max = max
}
func (this *MessageManager) saveToFile() {
	// 只有留言板才支持持久化
	if this.historyFileName != "" {
		messages := this.getMessage(0)
		history := new(HistoryMessages)
		history.Messages = make([]Message, len(messages))
		copy(history.Messages, messages)
		data, err := json.MarshalIndent(history, "", "    ")
		if err != nil {
			log.Println("[ERR]writeAdminCfg fail:", err)
			return
		}
		WriteConfig(this.historyFileName, data)
	}

}

func (this *MessageManager) appendMessage(line string) {
	this.messageListMutex.Lock()
	if this.messages.Len() > this.max {
		this.messages.Remove(this.messages.Front())
	}
	newLine := strings.TrimSpace(line)
	this.messageId += 1
	this.messages.PushBack(Message{this.messageId, newLine, getNowTime()})
	this.messageListMutex.Unlock()
	this.saveToFile()
}

func (this *MessageManager) getMessage(begin int) (ret []Message) {
	ret = make([]Message, 0)
	this.messageListMutex.RLock()
	if begin >= this.messageId {
		this.messageListMutex.RUnlock()
		return ret
	}
	for p := this.messages.Front(); p != nil; p = p.Next() {
		v := p.Value.(Message)
		if v.Id >= begin {
			ret = append(ret, v)
		}
	}
	this.messageListMutex.RUnlock()
	return ret
}

func (this *Mindustry) chartMessageProc(gameName string, message string) {
	this.chartMessages.appendMessage(message)
}

func (this *Mindustry) messageBoardProc(gameName string, message string) {
	this.messageBoard.appendMessage("[" + gameName + "] " + message)
	this.say("info.message_board", gameName)
}

type Admin struct {
	// web app username
	UserName string `json:"userName"`
	// game name
	Name         string `json:"name"`
	Id           string `json:"id"`
	Passwd       string `json:"passwd"`
	Contact      string `json:"contact"`
	LastVistTime string `json:"last vist time"`
	onlineTime   int64  `json:"online time"`
	sessionId    string `json:"sessionId"`
	level        int
}
type AdminCfg struct {
	SuperAdminList []Admin `json:"super admin"`
	AdminList      []Admin `json:"admin"`
	SignList       []Admin `json:"sign lst"`
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

type Assets struct {
	Name               string `json:"name"`
	UpdatedAt          string `json:"updated_at"`
	Size               int64  `json:"size"`
	BrowserDownloadUrl string `json:"browser_download_url"`
}

type GithubReleasesLatestApi struct {
	TagName     string   `json:"tag_name"`
	PublishedAt string   `json:"published_at"`
	AssetsList  []Assets `json:"assets"`
}

type MindustryVersionInfo struct {
	CurrVer       string                    `json:"curr_ver"`
	LocalFileName string                    `json:"local_file_name"`
	IsFixVer      bool                      `json:"is_fix_ver"`
	VerList       []GithubReleasesLatestApi `json:"ver_list"`
}

type GameStatus struct {
	CurrVer    string `json:"curr_ver"`
	NewVer     string `json:"new_ver"`
	ManagerVer string `json:"manager_ver"`
	CpuTemp    string `json:"cpuTemp"`
	Fps        string `json:"fps"`
	Online     string `json:"online"`
	Running    string `json:"running"`
	RunTime    string `json:"run_time"`
	Map        string `json:"map"`
	Wave       string `json:"wave"`
}

type User struct {
	Name         string `json:"name"`
	Uuid         string `json:"uuid"`
	OnlineTime   string `json:"online_time"`
	IsAdmin      bool   `json:"admin"`
	IsSuperAdmin bool   `json:"suop"`
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
	lastUsers              *linkedhashmap.Map
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
	firstIsStart           bool
	serverIsStart          bool
	serverIsRun            bool
	maps                   []string
	userCmdProcHandles     map[string]UserCmdProcHandle
	l                      *lingo.L
	i18n                   lingo.T
	currBanList            *BanCfg
	tmpBanList             *BanCfg
	m_isPermitMapModify    bool
	isShowDefaultMapInMaps bool
	mapMangePort           int
	maxMapCount            int
	c                      *cron.Cron
	cmdIn                  io.WriteCloser
	fpsInfo                string
	isInGameCmd            bool
	missionMap             string
	timeoutCnt             int
	killCh                 chan os.Signal
	cmdWaitTimer           *time.Timer
	isAutoRestartedForDay  bool
	isPlayerExistForHour   bool
	isPlayerExistForTenMin bool
	isNeedRestartForUpdate bool
	currMindustryVer       string
	mindustryVersionInfo   *MindustryVersionInfo
	gameStatus             *GameStatus
	startTime              int64
	chartMessages          *MessageManager
	messageBoard           *MessageManager
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
func (this *Mindustry) getUserId(name string) string {
	for uuid, user := range this.users {
		if name == user.Name {
			return uuid
		}
	}
	return ""
}

func findInBan(banCfgList []Ban, u *Ban) bool {
	for _, currBan := range banCfgList {
		if u.Id == currBan.Id && u.Name == currBan.Name && u.Ip == currBan.Ip {
			return true
		}
	}
	return false
}
func (this *Mindustry) loadBan() {
	data, err := ioutil.ReadFile("bans.json")
	if err != nil {
		log.Printf("[ERR]Not found bans.json!\n")
		return
	}
	err = json.Unmarshal(data, &this.currBanList)
	if err != nil {
		log.Printf("[ERR]Load banb cfg fail!\n")
	}
}

func (this *Mindustry) downloadUrl(remoteUrl string, localFileName string, size int64) bool {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{Transport: tr}

	resp, netErr := client.Get(remoteUrl)
	if netErr != nil {
		log.Printf("[ERR]Get remote info fail, remoteUrl:%s, netError:%v!\n", remoteUrl, netErr)
		return false
	}
	defer resp.Body.Close()

	out, err := os.Create(localFileName)
	if err != nil {
		log.Printf("[ERR]Get remote info os.Create fail, remoteUrl:%s, err:%v!\n", remoteUrl, err)
		return false
	}
	defer out.Close()
	copySize, err := io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("[ERR]Get remote info io.Copy fail, remoteUrl:%s, err:%v!\n", remoteUrl, err)
		return false
	}
	if copySize != size {
		log.Printf("[ERR]Get remote info io.Copy size invalid, remoteUrl:%s, copySize:%d, expSize:%d!\n", remoteUrl, copySize, size)
		return false
	}

	return true
}

func (this *Mindustry) downloadMindustryJar(remoteTagInfo *GithubReleasesLatestApi) string {

	for _, asset := range remoteTagInfo.AssetsList {
		index := strings.Index(asset.Name, "server")
		if !strings.HasSuffix(asset.Name, ".jar") || index < 0 {
			continue
		}
		localeFileName := remoteTagInfo.TagName + "_server-release.jar"
		if this.downloadUrl(asset.BrowserDownloadUrl, localeFileName, asset.Size) {
			return localeFileName
		}
		log.Printf("[ERR]download remote jar fail,BrowserDownloadUrl:%s, localeFileName:%s!\n", asset.BrowserDownloadUrl, localeFileName)
	}
	log.Printf("[ERR]not found remote server jar!\n")
	return ""
}

func (this *Mindustry) autoUpdateMindustryVer() {
	log.Printf("[INFO]autoUpdateMindustryVer check.\n")
	if this.mindustryVersionInfo.IsFixVer {
		log.Printf("[INFO]Not need update mindustry,IsFixVer is true!\n")
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(10 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*10)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
		},
	}
	request, netErr := http.NewRequest("GET", "https://api.github.com/repos/Anuken/Mindustry/releases/latest", nil)
	if netErr != nil {
		log.Printf("[ERR]Get remote mindustry version info fail,netError:%v!\n", netErr)
		return
	}
	response, responseErr := client.Do(request)
	if responseErr != nil {
		log.Printf("[ERR]Get remote mindustry version info rsp fail,netError:%v!\n", responseErr)
		return
	}
	if response.StatusCode == 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("[ERR]Get remote mindustry jar info ioutil.ReadAll fail!\n")
			return
		}
		var currRemoteReleasesLatest = GithubReleasesLatestApi{}
		err = json.Unmarshal(body, &currRemoteReleasesLatest)
		if err != nil {
			log.Printf("[ERR]Get remote mindustry jar info Unmarshal fail!\n")
			return
		}
		if currRemoteReleasesLatest.TagName == "" {
			log.Printf("[ERR]rempte mindustry version not found, don't need update!\n")
			return
		}
		if currRemoteReleasesLatest.TagName == this.mindustryVersionInfo.CurrVer {
			log.Printf("[INFO]curr mindustry version is lastest, don't need update!\n")
			return
		}
		localFileName := this.downloadMindustryJar(&currRemoteReleasesLatest)
		if localFileName == "" {
			log.Printf("[ERR]downloadMindustryJar fail!\n")
			return
		}
		this.mindustryVersionInfo.CurrVer = currRemoteReleasesLatest.TagName
		this.mindustryVersionInfo.LocalFileName = localFileName
		if len(this.mindustryVersionInfo.VerList) < 10 {
			this.mindustryVersionInfo.VerList = append(this.mindustryVersionInfo.VerList, currRemoteReleasesLatest)
		} else {
			remoteLastestTime, _ := time.Parse(time.RFC3339, currRemoteReleasesLatest.PublishedAt)
			minTime, _ := time.Parse(time.RFC3339, this.mindustryVersionInfo.VerList[0].PublishedAt)
			minIndex := 0
			for index, ver := range this.mindustryVersionInfo.VerList {
				t, _ := time.Parse(time.RFC3339, ver.PublishedAt)
				if remoteLastestTime.After(minTime) {
					minIndex = index
					minTime = t
				}
			}
			if remoteLastestTime.After(minTime) {
				this.mindustryVersionInfo.VerList[minIndex] = currRemoteReleasesLatest
			}
		}
		this.isNeedRestartForUpdate = true
		log.Printf("[INFO]downloadMindustryJar(%s) succ,wait restart server!\n", this.mindustryVersionInfo.CurrVer)
		this.writeMindustryVersionInfoCfg()
	} else {
		log.Printf("[ERR]Get remote mindustry version info fail,remote response:%d!\n", response.StatusCode)
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
		admin.level = 9
	}
	for _, admin := range this.adminCfg.AdminList {
		log.Printf("Admin:%s(%s)\n", admin.Name, admin.Id)
		admin.level = 1
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
			optionValue, err = cfg.String("server", "nameHead")
			if err == nil {
				nameHead := strings.TrimSpace(optionValue)
				this.name = nameHead + this.name
			} else {
				this.name = "[CIG]" + this.name
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

			optionValue, err = cfg.String("server", "isShowDefaultMapInMaps")
			if err == nil {
				this.isShowDefaultMapInMaps = strings.TrimSpace(optionValue) == "1"
				log.Printf("[ini]isShowDefaultMapInMaps:%t\n", this.isShowDefaultMapInMaps)
			}

			optionIntValue, errInt := cfg.Int("server", "mindustryPort")
			if errInt == nil {
				this.port = optionIntValue
				log.Printf("[ini]port:%d\n", this.port)
			}

			optionIntValue, err = cfg.Int("server", "mapMangePort")
			if err == nil {
				this.mapMangePort = optionIntValue
				log.Printf("[ini]mapMangePort:%d\n", this.mapMangePort)
			}

			optionIntValue, err = cfg.Int("server", "maxMapCount")
			if err == nil {
				this.maxMapCount = optionIntValue
				log.Printf("[ini]maxMapCount:%d\n", this.maxMapCount)
			}

			optionValue, err = cfg.String("server", "mode")
			if err == nil {
				if optionValue != "none" && optionValue != "" && checkMode(optionValue) {
					if checkMode(optionValue) {
						this.mode = optionValue
						log.Printf("[ini]fix mode:%s\n", this.mode)
					} else {
						log.Printf("[ini]invalid mode:%s\n", optionValue)
					}
				}
			}

		}
	}
	this.loadBan()
}

func (this *Mindustry) loadMindustryVersionInfoCfg() {
	this.mindustryVersionInfo.CurrVer = ""
	this.mindustryVersionInfo.LocalFileName = ""
	this.mindustryVersionInfo.IsFixVer = false
	data, err := ioutil.ReadFile("mindustryVersionCfg.json")
	if err != nil {
		log.Printf("[INFO]Not found mindustryVersionCfg.json, use jarPath in config.ini!\n")
		return
	}
	err = json.Unmarshal(data, this.mindustryVersionInfo)
	if err != nil {
		log.Printf("[ERR]Load cfg fail:mindustryVersionCfg.json, use jarPath in config.ini!\n!\n")
		return
	}

	log.Printf("mindustry curr ver:%s, IsFixVer:%t\n", this.mindustryVersionInfo.CurrVer, this.mindustryVersionInfo.IsFixVer)

	for _, ver := range this.mindustryVersionInfo.VerList {
		log.Printf("TagName:%s, PublishedAt:%s\n", ver.TagName, ver.PublishedAt)
	}
}

func (this *Mindustry) writeMindustryVersionInfoCfg() {
	data, err := json.MarshalIndent(this.mindustryVersionInfo, "", "    ")
	if err != nil {
		log.Println("[ERR]writeMindustryVersionInfoCfg fail:", err)
		return
	}
	WriteConfig("mindustryVersionCfg.json", data)
}

func (this *Mindustry) initStatus() {
	this.serverIsRun = false
	this.playCnt = 0
	this.timeoutCnt = 0
	this.m_isPermitMapModify = false
	this.fpsInfo = "UNKOWN"
	this.users = make(map[string]User)
	this.votetickUsers = make(map[string]int)
	this.isPlayerExistForHour = false
	this.isPlayerExistForTenMin = false
	this.isAutoRestartedForDay = true
	this.isNeedRestartForUpdate = false
}
func (this *Mindustry) init() {
	this.chartMessages = new(MessageManager)
	this.chartMessages.init(MAX_MESSAGE_CACHE)
	this.messageBoard = new(MessageManager)
	this.messageBoard.init(MAX_MESSAGE_CACHE)
	this.messageBoard.loadHistory("message_board.json")
	this.serverOutR, _ = regexp.Compile(".*(\\[INFO\\]|\\[ERR\\])(.*)")
	this.cmds = make(map[string]Cmd)
	this.cmdHelps = make(map[string]string)
	this.userCmdProcHandles = make(map[string]UserCmdProcHandle)
	rand.Seed(time.Now().UnixNano())
	this.name = fmt.Sprintf("mindustry-%d", rand.Int())
	this.jarPath = "server-release.jar"
	this.currMindustryVer = ""
	this.mindustryVersionInfo = new(MindustryVersionInfo)
	this.missionMap = "nuclearProductionComplex"
	this.firstIsStart = true
	this.serverIsStart = false
	this.c = cron.New()
	this.adminCfg = new(AdminCfg)
	this.tmpBanList = new(BanCfg)
	this.currBanList = new(BanCfg)
	this.gameStatus = new(GameStatus)
	this.currBanList.BanList = make([]Ban, 0)
	this.adminCfg.SignList = make([]Admin, 0)
	this.adminCfg.AdminList = make([]Admin, 0)
	this.lastUsers = linkedhashmap.New()
	this.port = 6567
	this.mapMangePort = 6569
	this.maxMapCount = 15
	this.startTime = time.Now().Unix()
	this.killCh = make(chan os.Signal)
	this.loadConfig()
	this.loadAdminConfig()
	this.loadMindustryVersionInfoCfg()
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
	this.userCmdProcHandles["vote"] = this.proc_votetick
	this.userCmdProcHandles["mode"] = this.proc_mode
	this.userCmdProcHandles["mapManage"] = this.proc_mapManage
	this.initStatus()
	spec := "0 0 * * * ?"
	this.c.AddFunc(spec, func() {
		this.hourTask()
	})
	spec = "0 5/10 * * * ?"
	this.c.AddFunc(spec, func() {
		this.tenMinTask()
	})
	this.c.Start()
}

var colorCodeReg = regexp.MustCompile(`\[.*?\]|#[0-9a-fA-F]+`)

func removeColorCode(str string) string {
	newStr := colorCodeReg.ReplaceAllString(str, "")
	return newStr
}

func (this *Mindustry) execCommand(commandName string, params []string) error {
	cmd := exec.Command(commandName, params...)
	log.Println(cmd.Args)
	stdout, outErr := cmd.StdoutPipe()
	stderr, errErr := cmd.StderrPipe()
	if outErr != nil {
		return outErr
	}
	if errErr != nil {
		return errErr
	}

	var inErr error
	this.cmdIn, inErr = cmd.StdinPipe()
	if inErr != nil {
		return inErr
	}
	cmd.Start()
	this.initStatus()
	go func(cmd *exec.Cmd) {
		this.killCh = make(chan os.Signal)
		signal.Notify(this.killCh, os.Interrupt, os.Kill)
		s := <-this.killCh
		if cmd.Process != nil {
			log.Printf("sub process exit:%s", s)
			cmd.Process.Kill()
		}
	}(cmd)
	this.c.Start()
	go func() {
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
			this.execCmd(inputCmd)
		}
	}()

	go func() {
		reader := bufio.NewReader(stdout)

		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			fmt.Printf(line)
			this.output(StripColor(line))
		}
	}()
	go func() {
		readerErr := bufio.NewReader(stderr)
		for {
			line, err2 := readerErr.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			log.Printf(line)
			if strings.Contains(line, "java.lang.RuntimeException") {
				log.Printf("server crash, force reboot!\n")
				this.serverIsRun = false
			}
		}
	}()
	cmd.Wait()
	return nil
}
func (this *Mindustry) hourTask() {
	hour := time.Now().Hour()
	log.Printf("hourTask trig:%d\n", hour)
	isAutoRestartForDay := false
	if hour >= 2 && hour <= 5 {
		isAutoRestartForDay = true
	} else {
		this.isAutoRestartedForDay = false
	}
	this.autoUpdateMindustryVer()
	if isAutoRestartForDay && !this.isAutoRestartedForDay && !this.isPlayerExistForHour {
		log.Printf("game is auto restar for not player!\n")
		this.isAutoRestartedForDay = true
		this.execCmd("exit")
		return
	}
	this.isPlayerExistForHour = false
	if this.serverIsRun {
		this.execCmd("save " + strconv.Itoa(hour))
		this.say("info.auto_save", hour)
		this.execCmd("bans")
	} else {
		log.Printf("game is not running.\n")
	}
}

func (this *Mindustry) tenMinTask() {
	log.Printf("tenMinTask trig[%.3f°C].\n", getCpuTemp())

	if !this.serverIsStart {
		return
	}
	if this.timeoutCnt >= 3 {
		log.Printf("game is not response,exit.\n")
		this.killCh <- os.Kill
	} else if !this.serverIsRun {
		log.Printf("game is not running,exit.\n")
		this.execCmd("exit")
	} else if this.isNeedRestartForUpdate && !this.isPlayerExistForTenMin {
		log.Printf("game is need update, exit.\n")
		this.execCmd("exit")
	} else {
		this.say(this.notice)
		if this.currProcCmd != "" {
			log.Printf("cmd(%s) is running.\n", this.currProcCmd)
		} else {
			log.Printf("update game status.\n")
			this.isInGameCmd = false
			this.proc_status("", "Server", "status", false)
		}
	}
	this.isPlayerExistForTenMin = false
}

var userMutex sync.RWMutex
var lastUserMutex sync.RWMutex

const MAX_LAST_USERS int = 500

func (this *Mindustry) addUser(name string, uuid string, isAdmin bool, isSuop bool) {
	u := User{name, uuid, getNowTime(), isAdmin, isSuop, 0}
	if _, ok := this.users[uuid]; ok {
		return
	}
	userMutex.Lock()
	this.users[uuid] = u
	userMutex.Unlock()

	lastUserMutex.Lock()
	_, isExist := this.lastUsers.Get(uuid)
	if !isExist && this.lastUsers.Size() >= MAX_LAST_USERS {
		it := this.lastUsers.Iterator()
		for it.Next() {
			key := it.Key().(string)
			if _, ok := this.users[key]; !ok {
				this.lastUsers.Remove(key)
				break
			}
		}
	}
	this.lastUsers.Put(uuid, u)
	lastUserMutex.Unlock()

	this.playCnt++
	log.Printf("add user info :%s,playCnt:%d\n", name, this.playCnt)
}
func (this *Mindustry) getUserListForWeb(isOnline bool) (ret []User) {
	ret = make([]User, 0)
	if isOnline {
		userMutex.RLock()
		for _, user := range this.users {
			ret = append(ret, user)
		}
		userMutex.RUnlock()
	} else {
		lastUserMutex.RLock()
		it := this.lastUsers.Iterator()
		for it.Next() {
			ret = append(ret, it.Value().(User))
		}
		lastUserMutex.RUnlock()
	}

	return ret
}

func (this *Mindustry) onlineAdmin(uuid string) {
	if _, ok := this.users[uuid]; !ok {
		log.Printf("user %s not found\n", this.users[uuid].Name)
		return
	}
	tempUser := this.users[uuid]
	tempUser.IsAdmin = true
	if tempUser.level < 1 {
		tempUser.level = 1
	}
	this.users[uuid] = tempUser
	log.Printf("online admin :%s\n", this.users[uuid].Name)
}

func (this *Mindustry) onlineSuperAdmin(uuid string) {
	if _, ok := this.users[uuid]; !ok {
		log.Printf("user %s not found\n", this.users[uuid].Name)
		return
	}
	tempUser := this.users[uuid]
	tempUser.IsAdmin = true
	tempUser.IsSuperAdmin = true
	if tempUser.level < 9 {
		tempUser.level = 9
	}
	this.users[uuid] = tempUser
	log.Printf("online superAdmin :%s\n", this.users[uuid].Name)
}

func getNowDate() string {
	return time.Now().Format("06-01-02")
}

func getNowTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
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
	if _, ok := this.users[uuid]; ok {
		return
	}
	this.isPlayerExistForHour = true
	this.isPlayerExistForTenMin = true
	role := this.getUserRole(name, uuid)
	this.addUser(name, uuid, role == ADMIN, role == SUPER_ADMIN)
	if role == SUPER_ADMIN {
		this.onlineSuperAdmin(uuid)
	} else if role == ADMIN {
		this.onlineAdmin(uuid)
	}
}
func (this *Mindustry) offlineAllUser() {
	this.playCnt = 0
	for uuid, user := range this.users {
		this.offlineUser(user.Name, uuid)
	}
}
func (this *Mindustry) offlineUser(name string, uuid string) {
	if _, ok := this.users[uuid]; !ok {
		return
	}

	this.delUser(name, uuid)
	return
}
func (this *Mindustry) delUser(name string, uuid string) {
	if _, ok := this.users[uuid]; !ok {
		log.Printf("del user not exist :%s\n", name)
		return
	}
	if this.playCnt > 0 {
		this.playCnt--
	}
	userMutex.Lock()
	delete(this.users, uuid)
	userMutex.Unlock()
	lastUserMutex.Lock()
	oldU, isExist := this.lastUsers.Get(uuid)
	if !isExist {
		log.Printf("last user not exist :%s, uuid:%s\n", name, uuid)
		lastUserMutex.Unlock()
	}
	updateUser := oldU.(User)
	updateUser.OnlineTime = getNowTime()
	this.lastUsers.Put(uuid, updateUser)
	lastUserMutex.Unlock()

	log.Printf("del user info :%s,playCnt:%d\n", name, this.playCnt)
}
func (this *Mindustry) execCmd(cmd string) {
	if cmd == "stop" || cmd == "host" || cmd == "hostx" || cmd == "load" {
		this.offlineAllUser()
	}
	log.Printf("execCmd :%s\n", cmd)
	data := []byte(cmd + "\n")
	this.cmdIn.Write(data)
}
func (this *Mindustry) say(strKey string, v ...interface{}) {
	localeStr := "say " + this.i18n.Value(strKey) + "\n"
	info := fmt.Sprintf(localeStr, v...)
	this.cmdIn.Write([]byte(info))
}
func (this *Mindustry) jsToast(showTime int, strKey string, v ...interface{}) {
	localeStr := this.i18n.Value(strKey)
	info := fmt.Sprintf(localeStr, v...)
	info = "js Call.infoToast(\"" + info + "\"," + strconv.Itoa(showTime) + ")\n"
	this.cmdIn.Write([]byte(info))
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
func (this *Mindustry) cmdWaitTimeout(uuid string, userInput string, cmdName string) {
	go func() {
		this.cmdWaitTimer = time.NewTimer(time.Duration(5) * time.Second)
		<-this.cmdWaitTimer.C
		if this.currProcCmd != "" {
			log.Printf("cmd:%s timeout!\n", this.currProcCmd)
			this.timeoutCnt++
			if this.isInGameCmd {
				this.say("error.cmd_timeout", this.currProcCmd)
			}
			this.currProcCmd = ""
		}
	}()
	this.currProcCmd = cmdName
}

func (this *Mindustry) proc_maps(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.cmdWaitTimeout(uuid, userInput, "maps")
	this.execCmd("reloadmaps")
	this.maps = this.maps[0:0]
	this.execCmd("maps")
	return true
}
func (this *Mindustry) proc_status(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.cmdWaitTimeout(uuid, userInput, "status")
	this.execCmd("status")

	return true
}
func (this *Mindustry) proc_host(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	mapName := ""
	temps := strings.Split(userInput, " ")
	if len(temps) < 2 {
		this.say("error.cmd_length_invalid", userInput)
		return false
	}
	inputCmd := strings.TrimSpace(temps[0])
	inputMap := strings.TrimSpace(temps[1])
	inputMode := ""
	if this.mode != "" {
		if len(temps) > 2 {
			this.say("error.cmd_host_fix_mode", this.mode)
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
			this.say("error.cmd_hostx_id_not_number", userInput)
			return false
		}
		if inputIndex < 0 || inputIndex >= len(this.maps) {

			this.say("error.cmd_hostx_id_not_found", userInput)
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
			this.say("error.cmd_host_map_not_found", userInput)
			return false
		}
	} else {
		this.say("error.cmd_invalid", userInput)
		return false
	}
	if !checkMode(inputMode) {
		this.say("error.cmd_host_mode_invalid", userInput)
		return false
	}
	if isOnlyCheck {
		return true
	}
	if inputMode == "mission" {
		this.missionMap = mapName
		this.serverIsRun = false
		this.execCmd("exit")
		return true
	}
	this.execCmd("reloadmaps")
	time.Sleep(time.Duration(5) * time.Second)
	mapName = strings.Replace(mapName, " ", "_", -1)
	if inputMode == "" {
		this.execCmd("nextmap " + mapName)
		this.execCmd("gameover")
	} else {
		this.say("info.server_restart")
		this.execCmd("stop")
		time.Sleep(time.Duration(5) * time.Second)
		this.execCmd("host " + mapName + " " + inputMode)
	}
	return true
}

func (this *Mindustry) proc_save(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	targetSlot := ""
	if userInput == "save" {
		minute := time.Now().Minute()
		targetSlot = fmt.Sprintf("%d%02d%02d", time.Now().Day(), time.Now().Hour(), minute/10*10)
	} else {
		targetSlot = userInput[len("save"):]
		targetSlot = strings.TrimSpace(targetSlot)
	}
	if _, ok := strconv.Atoi(targetSlot); ok != nil {
		this.say("error.cmd_save_slot_invalid", targetSlot)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.execCmd("save " + targetSlot)
	this.say("info.save_slot_succ", targetSlot)
	return true
}

func (this *Mindustry) proc_load(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	targetSlot := userInput[len("load"):]
	targetSlot = strings.TrimSpace(targetSlot)
	if !checkSlotValid(targetSlot) {
		this.say("error.cmd_load_slot_invalid", targetSlot)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.say("info.server_restart")
	time.Sleep(time.Duration(5) * time.Second)
	this.execCmd("stop")
	time.Sleep(time.Duration(5) * time.Second)
	this.execCmd(userInput)
	return true
}

func (this *Mindustry) getAdmin(userName string) *Admin {
	for i, admin := range this.adminCfg.SuperAdminList {
		if admin.UserName == userName {
			return &this.adminCfg.SuperAdminList[i]
		}
	}
	for i, admin := range this.adminCfg.AdminList {
		if admin.UserName == userName {
			return &this.adminCfg.AdminList[i]
		}
	}

	return nil
}

func (this *Mindustry) getAdminByGameName(gameName string) *Admin {
	for i, admin := range this.adminCfg.SuperAdminList {
		if admin.Name == gameName {
			return &this.adminCfg.SuperAdminList[i]
		}
	}
	for i, admin := range this.adminCfg.AdminList {
		if admin.Name == gameName {
			return &this.adminCfg.AdminList[i]
		}
	}

	return nil
}

func (this *Mindustry) getSign(userName string) *Admin {
	for i, admin := range this.adminCfg.SignList {
		if admin.UserName == userName {
			return &this.adminCfg.SignList[i]
		}
	}

	return nil
}

func (this *Mindustry) getSignByGameName(gameName string) *Admin {
	for i, admin := range this.adminCfg.SignList {
		if admin.Name == gameName {
			return &this.adminCfg.SignList[i]
		}
	}

	return nil
}

func (this *Mindustry) denySign(userName string) bool {
	for i, admin := range this.adminCfg.SignList {
		if admin.UserName == userName {
			gameName := admin.Name
			this.adminCfg.SignList = append(this.adminCfg.SignList[:i], this.adminCfg.SignList[i+1:]...)
			this.writeAdminConfig()
			log.Printf("[INFO]rmv sign,userName=%s,gameName=%s\n", userName, gameName)
			return true
		}
	}
	log.Printf("[INFO]deny sign,i not found, userName=%s\n", userName)
	return false
}

func (this *Mindustry) addAdmin(userName string) bool {
	signAdmin := this.getSign(userName)
	if signAdmin == nil {
		log.Printf("[INFO]addAdmin,not found, userName=%s\n", userName)
		return false
	}
	newAdmin := *signAdmin
	this.adminCfg.AdminList = append(this.adminCfg.AdminList, newAdmin)
	this.denySign(userName)
	this.writeAdminConfig()
	log.Printf("[INFO]add admin,userName=%s\n", userName)
	return true
}

func (this *Mindustry) regAdmin(userName string, gameName string, passwd string, contact string) bool {
	if this.getAdmin(userName) != nil || this.getAdminByGameName(gameName) != nil {
		log.Printf("[INFO]regAdmin,is exist in admin list, userName=%s,gameName=%s\n", userName, gameName)
		return false
	}
	if this.getSign(userName) != nil || this.getSignByGameName(gameName) != nil {
		log.Printf("[INFO]regAdmin,is exist in sign list, userName=%s,gameName=%s\n", userName, gameName)
		return false
	}

	cryPasswd := cryptoMd5(passwd)
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	newAdmin := Admin{userName, gameName, "", cryPasswd, contact, nowTime, 0, "", 1}
	this.adminCfg.SignList = append(this.adminCfg.SignList, newAdmin)
	this.writeAdminConfig()
	log.Printf("[INFO]regAdmin,userName=%s,gameName=%s\n", userName, gameName)
	return true
}

func (this *Mindustry) rmvAdmin(userName string) bool {
	for i, admin := range this.adminCfg.AdminList {
		if admin.UserName == userName {
			gameName := admin.Name
			this.adminCfg.AdminList = append(this.adminCfg.AdminList[:i], this.adminCfg.AdminList[i+1:]...)
			this.writeAdminConfig()
			log.Printf("[INFO]rmvAdmin,userName=%s,gameName=%s\n", userName, gameName)
			return true
		}
	}
	log.Printf("[INFO]rmvAdmin,i not found, userName=%s\n", userName)
	return false
}

func (this *Mindustry) webLoginAdmin(userName string, passwd string) bool {
	admin := this.getAdmin(userName)
	if admin == nil {
		log.Printf("[INFO]web login,not found userName=%s", userName)
		return false
	}
	if admin.Passwd != cryptoMd5(passwd) {
		log.Printf("[INFO]web login,passwd err, userName=%s\n", userName)
		return false
	}
	gameName := admin.Name
	sessionIdTmp := time.Now().UnixNano() * rand.Int63()
	if sessionIdTmp < 0 {
		sessionIdTmp *= -1
	}
	admin.sessionId = strconv.FormatInt(sessionIdTmp, 10)
	log.Printf("[INFO]web login,userName=%s,gameName=%s, sessionId=%s\n", userName, gameName, admin.sessionId)
	return true
}
func (this *Mindustry) webLoginSessionChk(userName string, sessionId string) bool {
	admin := this.getAdmin(userName)
	if admin == nil {
		log.Printf("[INFO]web login,not found userName=%s", userName)
		return false
	}

	if admin.sessionId == sessionId {
		admin.onlineTime = time.Now().Unix()
		return true
	}
	//log.Printf("[INFO]web login,not found session, userName=%s, sessionId=%s", userName, sessionId)
	return false
}

func (this *Mindustry) webLoginUuidReset(userName string, gameName string) bool {
	admin := this.getAdmin(userName)
	if admin == nil {
		log.Printf("[INFO]uuid reset,not found userName=%s", userName)
		return false
	}

	admin.Name = gameName
	admin.Id = ""
	this.writeAdminConfig()
	log.Printf("[INFO]web login,reset uuid, userName=%s", userName)
	return true
}

func (this *Mindustry) webLoginModifyPasswd(userName string, passwd string) bool {
	admin := this.getAdmin(userName)
	if admin == nil {
		log.Printf("[INFO]web login modifyPasswd,not found userName=%s", userName)
		return false
	}

	admin.Passwd = cryptoMd5(passwd)
	this.writeAdminConfig()
	log.Printf("[INFO]passwd modified, userName=%s", userName)
	return true
}

func (this *Mindustry) webLoginIsSop(userName string) bool {
	for _, admin := range this.adminCfg.SuperAdminList {
		if admin.UserName == userName {
			return true
		}
	}
	return false
}

func (this *Mindustry) proc_directCmd(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.execCmd(userInput)
	return true
}
func (this *Mindustry) proc_gameover(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.execCmd("reloadmaps")
	this.execCmd(userInput)
	return true
}
func (this *Mindustry) proc_help(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	temps := strings.Split(userInput, " ")
	if len(temps) >= 2 {
		cmd := strings.TrimSpace(temps[1])
		this.say("helps."+cmd, cmd)
	} else {
		if this.isSuop(uuid, userName) {
			this.say("info.super_admin_cmd", this.cfgSuperAdminCmds)
		} else if this.users[uuid].IsAdmin {
			this.say("info.admin_cmd", this.cfgAdminCmds)
		} else {
			this.say("info.user_cmd", this.cfgNormCmds)
		}
		this.say("info.votetick_cmd", this.cfgVoteCmds)

	}
	return true
}

var tempOsPath = "/sys/class/thermal/thermal_zone0/temp"
var tempOsPath_linux_amd64 = "/sys/class/thermal/thermal_zone1/temp"

func getCpuTemp() float64 {
	tempFile := tempOsPath
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		tempFile = tempOsPath_linux_amd64
	}
	raw, err := ioutil.ReadFile(tempFile)
	if err != nil {
		//log.Printf("Failed to read temperature from %q: %v", tempFile, err)
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
func (this *Mindustry) proc_show(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.proc_status(uuid, userName, userInput, false)

	return true
}
func (this *Mindustry) proc_admins(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	isShowWarn := this.isSuop(uuid, userName)
	this.say("info.super_admin_list", this.getAdminList(this.adminCfg.SuperAdminList, isShowWarn))
	this.say("info.admin_list", this.getAdminList(this.adminCfg.AdminList, isShowWarn))
	return true

}

func (this *Mindustry) proc_slots(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if isOnlyCheck {
		return true
	}
	this.say("info.slots_list", getSlotList())
	return true
}
func (this *Mindustry) checkVote() (bool, int, int) {
	if this.playCnt == 0 {
		log.Printf("playCnt is zero!\n")
		return false, 0, 0
	}
	agreeCnt := 0
	againstCnt := 0
	for uuid, isAgree := range this.votetickUsers {
		if _, ok := this.users[uuid]; !ok {
			continue
		}
		if isAgree == 1 {
			agreeCnt++
		} else {
			againstCnt++
		}
	}
	notVoteUserCnt := 0
	for uuid, _ := range this.users {
		if _, ok := this.votetickUsers[uuid]; !ok {
			notVoteUserCnt++
		}
	}
	againstPoint := float32(againstCnt)*1 + float32(notVoteUserCnt)*0.5
	agreePoint := float32(agreeCnt) * 1
	return agreePoint/(agreePoint+againstPoint) >= 0.5, agreeCnt, againstCnt
}
func (this *Mindustry) proc_votetick(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	index := strings.Index(userInput, " ")
	if index < 0 {
		this.say("error.cmd_votetick_target_invalid", userInput)
		return false
	}

	if len(this.votetickUsers) > 0 {
		this.say("error.cmd_votetick_in_progress")
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
			this.say("error.cmd_votetick_not_permit", votetickCmdHead)
			return false
		}
	} else {
		this.say("error.cmd_votetick_cmd_error", votetickCmdHead)
		return false
	}
	handleFunc := this.proc_directCmd
	if tempHandleFunc, ok := this.userCmdProcHandles[votetickCmdHead]; ok {
		handleFunc = tempHandleFunc
	}
	checkRslt := handleFunc(uuid, userName, votetickCmd, true)
	if !checkRslt {
		log.Printf("precheck vote cmd [%s] fail\n", votetickCmd)
		return false
	}
	if isOnlyCheck {
		return true
	}

	this.currProcCmd = "votetick"
	this.votetickUsers = make(map[string]int)
	this.votetickUsers[uuid] = 1
	go func() {
		timer := time.NewTimer(time.Duration(60) * time.Second)
		<-timer.C
		isSucc, agreeCnt, againstCnt := this.checkVote()
		if isSucc {
			this.say("info.votetick_pass", this.playCnt, agreeCnt)
			handleFunc(uuid, userName, votetickCmd, false)
		} else {
			this.say("info.votetick_fail", this.playCnt, agreeCnt, againstCnt)
		}
		this.votetickUsers = make(map[string]int)
		this.currProcCmd = ""
	}()

	this.jsToast(58, "info.votetick_begin_info", votetickCmd)
	return true
}
func checkMode(inputMode string) bool {
	if inputMode != "mission" && inputMode != "pvp" && inputMode != "attack" && inputMode != "" && inputMode != "sandbox" && inputMode != "survival" {
		return false
	}
	return true
}
func (this *Mindustry) proc_mode(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	temps := strings.Split(userInput, " ")
	if len(temps) < 2 {
		this.say("info.mode_show", this.mode)
		return false
	}
	inputMode := temps[1]
	if inputMode == "none" {
		inputMode = ""
	}
	if inputMode == "mission" {
		this.say("info.mode_show", this.mode)
		return false
	}

	if !checkMode(inputMode) {
		this.say("error.cmd_host_mode_invalid", userInput)
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.mode = inputMode
	this.say("info.mode_show", this.mode)
	return true
}

func (this *Mindustry) proc_mapManage(uuid string, userName string, userInput string, isOnlyCheck bool) bool {
	if this.m_isPermitMapModify {
		this.say("error.cmd_mapManage_started")
		return false
	}
	if isOnlyCheck {
		return true
	}
	this.say("info.cmd_mapManage_start")
	this.m_isPermitMapModify = true
	go func() {
		timer := time.NewTimer(time.Duration(600) * time.Second)
		<-timer.C
		this.m_isPermitMapModify = false
		this.say("info.cmd_mapManage_end")
	}()
	return true
}
func (this *Mindustry) getUserLevel(uuid string, userName string) int {
	if userName != "" {
		admin := this.getAdmin(userName)
		if admin != nil {
			return admin.level
		} else {
			log.Printf("admin not found [%s]\n", userName)
			return 0
		}
	}
	return this.users[uuid].level
}
func (this *Mindustry) getGameName(uuid string, userName string) string {
	if userName != "" {
		admin := this.getAdmin(userName)
		if admin != nil {
			return admin.Name
		} else {
			log.Printf("admin not found [%s]\n", userName)
			return ""
		}
	}
	return this.users[uuid].Name
}
func (this *Mindustry) isSuop(uuid string, userName string) bool {
	if userName != "" {
		admin := this.getAdmin(userName)
		if admin != nil {
			return admin.level == 9
		} else {
			log.Printf("admin not found [%s]\n", userName)
			return false
		}
	}
	return this.users[uuid].IsSuperAdmin
}

func (this *Mindustry) procUsrCmd(uuid string, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	cmdName := temps[0]
	if userName != "" {
		log.Printf("admin [%s] exec cmd [%s]\n", userName, userInput)
	} else {
		log.Printf("user [%s] exec cmd [%s]\n", this.users[uuid].Name, userInput)
	}
	if cmd, ok := this.cmds[cmdName]; ok {
		if this.getUserLevel(uuid, userName) < cmd.level {
			this.say("error.cmd_permission_denied", cmdName)
			return
		} else {
			if this.currProcCmd != "" {
				this.say("error.cmd_is_exceuting", this.currProcCmd)
				return
			}
			this.isInGameCmd = true
			if handleFunc, ok := this.userCmdProcHandles[cmdName]; ok {
				handleFunc(uuid, userName, userInput, false)
			} else {
				this.userCmdProcHandles["directCmd"](uuid, userName, userInput, false)
			}
		}

	} else {
		this.say("error.cmd_invalid_user", cmdName)
	}
}
func (this *Mindustry) showStatus() {
	this.say("info.ver", _VERSION_)
	this.say("curr mindustry version:%s, new mindustry version:%s", this.currMindustryVer, this.mindustryVersionInfo.CurrVer)
	this.say("info.cpu_temperature", getCpuTemp())
	this.say("info.status_show", this.fpsInfo)
}

var lastUpdateTime int64 = 0

func (this *Mindustry) updateStatus() {
	currTime := time.Now().Unix()
	if currTime-10 < lastUpdateTime {
		return
	}
	this.gameStatus.NewVer = this.mindustryVersionInfo.CurrVer
	this.gameStatus.CurrVer = this.currMindustryVer
	this.gameStatus.ManagerVer = _VERSION_
	this.gameStatus.CpuTemp = strconv.FormatFloat(getCpuTemp(), 'f', 2, 64) + "℃"
	this.gameStatus.Fps = this.fpsInfo
	this.gameStatus.Running = strconv.FormatBool(this.serverIsRun)
	this.gameStatus.Online = strconv.Itoa(this.playCnt)
	runTime := (float64(currTime) - float64(this.startTime)) / 3600
	this.gameStatus.RunTime = strconv.FormatFloat(runTime, 'f', 2, 64) + "h"
}

func (this *Mindustry) multiLineRsltCmdComplete(line string) bool {
	index := -1
	if this.currProcCmd == "maps" {
		if strings.Index(line, "Map directory:") >= 0 {
			mapsInfo := "MAX([red]" + strconv.Itoa(this.maxMapCount) + "[])"
			for index, name := range this.maps {
				mapsInfo += " "
				mapsInfo += ("[cyan](" + strconv.Itoa(index) + ")[white]" + name)
			}
			this.say("info.maps_list", mapsInfo)
			this.timeoutCnt = 0
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
		//"   34 FPS, 22 MB used."
		if strings.Index(line, "FPS") >= 0 && strings.Index(line, "MB used.") >= 0 {
			this.fpsInfo = strings.TrimSpace(line)
			this.timeoutCnt = 0
		}
		index = strings.Index(line, "Players:")
		if index >= 0 {
			countStr := strings.TrimSpace(line[index+len("Players:")+1:])
			if count, ok := strconv.Atoi(countStr); ok == nil {
				this.playCnt = count
				log.Printf("curr playCnt:%d\n", this.playCnt)
			}
			this.isPlayerExistForHour = this.playCnt > 0
			this.isPlayerExistForTenMin = this.playCnt > 0

			if this.isInGameCmd {
				this.showStatus()
			}
			return true
		} else if strings.Index(line, "No players connected.") >= 0 {
			this.offlineAllUser()
			this.showStatus()
			this.timeoutCnt = 0
			return true
		} else if strings.Index(line, "Status: server closed") >= 0 {
			this.serverIsRun = false
			this.offlineAllUser()
			this.timeoutCnt = 0
			this.showStatus()
			return true
		}
	}
	return false
}

const USER_CONNECTED_KEY string = " has connected."
const USER_DISCONNECTED_KEY string = " has disconnected."
const SERVER_INFO_LOG string = "[I] "
const SERVER_ERR_LOG string = "[E] "
const SERVER_READY_KEY string = "Server loaded. Type 'help' for help."
const SERVER_STSRT_KEY string = "Opened a server on port"

var BAN_REG_ID = regexp.MustCompile("(.+?)\\s/\\sLast\\sknown\\sname\\:\\s'(.+?)'")
var BAN_REG_IP_ID = regexp.MustCompile("'(.+?)'\\s/\\sLast\\sknown\\sname\\:\\s'(.+?)'\\s/\\sID:\\s'(.+?)'")
var BAN_REG_IP = regexp.MustCompile("'(.+?)'\\s\\(No\\sknown\\sname\\sor\\sinfo\\)")

var BAN_REG_UNBAN = regexp.MustCompile("Unbanned\\splayer\\:\\s(.+)")
var BAN_REG_BAN = regexp.MustCompile("(.+?)\\shas\\sbanned\\s(.+)\\.")

func getUserByOutput(key string, cmdBody string) (string, string, bool) {
	index := strings.Index(cmdBody, key)
	if index < 0 {
		return "", "", false
	}

	userName := strings.TrimSpace(cmdBody[:index])
	uuidInfo := strings.TrimSpace(cmdBody[index:])
	lIndex := strings.Index(uuidInfo, "[")
	rIndex := strings.LastIndex(uuidInfo, "]")
	if lIndex < 0 || rIndex < 0 {
		return "", "", false
	}
	uuid := strings.TrimSpace(uuidInfo[lIndex+1 : rIndex])
	return userName, uuid, true
}

var banUpMutex sync.RWMutex

func (this *Mindustry) banUser(adminName string, username string) {
	banUpMutex.Lock()
	uuid := this.getUserId(username)
	this.currBanList.BanList = append(this.currBanList.BanList, Ban{username, uuid, "", adminName, getNowTime()})
	banUpMutex.Unlock()
	log.Printf("BanUser:%s->%s\n", adminName, username)
}

func (this *Mindustry) unbanUser(target string) {
	banUpMutex.Lock()
	j := 0
	for _, user := range this.currBanList.BanList {
		if user.Id != target && user.Name != target && user.Ip != target {
			this.currBanList.BanList[j] = user
			j++
		} else {
			log.Printf("UnbanUser:%s,%s,%s\n", user.Id, user.Name, user.Ip)
		}
	}
	this.currBanList.BanList = this.currBanList.BanList[:j]
	banUpMutex.Unlock()
}

func (this *Mindustry) updateBans() {
	banUpMutex.Lock()
	isChg := false
	j := 0
	for _, user := range this.currBanList.BanList {
		if !findInBan(this.tmpBanList.BanList, &user) {
			this.currBanList.BanList[j] = user
			j++
		} else {
			isChg = true
			log.Printf("rmv banUser:%s,%s,%s\n", user.Id, user.Name, user.Ip)
		}
	}
	this.currBanList.BanList = this.currBanList.BanList[:j]
	for _, user := range this.tmpBanList.BanList {
		if !findInBan(this.currBanList.BanList, &user) {
			this.currBanList.BanList = append(this.currBanList.BanList, user)
			isChg = true
			log.Printf("add banUser:%s,%s,%s\n", user.Id, user.Name, user.Ip)
		}
	}
	banUpMutex.Unlock()
	if isChg {
		data, err := json.MarshalIndent(this.currBanList, "", "    ")
		if err != nil {
			log.Println("[ERR]writeAdminCfg fail:", err)
			return
		}
		WriteConfig("bans.json", data)
	}
}

func (this *Mindustry) getBans() (ret []Ban) {
	banUpMutex.RLock()
	ret = make([]Ban, len(this.currBanList.BanList))
	copy(ret, this.currBanList.BanList)
	banUpMutex.RUnlock()
	return ret
}

func (this *Mindustry) webMessageProc(userName string, message string) {
	this.say("(" + userName + ") " + message)
	if !strings.HasPrefix(message, "\\") && !strings.HasPrefix(message, "/") && !strings.HasPrefix(message, "!") {
		return
	}
	this.procUsrCmd("", userName, message[1:])
}

var MAP_WAVE_REG = regexp.MustCompile("Playing\\son\\smap\\s(.+)/\\sWave\\s(\\d+)")
var MESSAGE_REG = regexp.MustCompile("<(.+)\\:\\s/message\\s(.+)>")

func (this *Mindustry) output(line string) {
	index := strings.Index(line, SERVER_ERR_LOG)
	if index >= 0 {
		errInfo := strings.TrimSpace(line[index+len(SERVER_ERR_LOG):])
		if strings.Contains(errInfo, "io.anuke.arc.util.ArcRuntimeException: File not found") {
			log.Printf("map not found , force exit!\n")
			this.execCmd("exit")
		}
		this.cmdFailReason = errInfo
		return
	}

	index = strings.Index(line, SERVER_INFO_LOG)
	if index < 0 {
		return
	}
	cmdBody := strings.TrimSpace(line[index+len(SERVER_INFO_LOG):])
	index = strings.Index(cmdBody, ":")
	if index > -1 {
		userName := strings.TrimSpace(cmdBody[:index])
		if userName == "Server" || this.getUserId(userName) != "" {
			this.chartMessageProc(userName, line)
		}
	}
	messageBoard := MESSAGE_REG.FindStringSubmatch(cmdBody)
	if messageBoard != nil {
		userName := messageBoard[1]
		log.Printf("[%s]message:%s\n", userName, messageBoard[2])
		if this.getUserId(userName) != "" {

			fmt.Printf("%s : %s\n", userName, messageBoard[2])
			this.messageBoardProc(userName, messageBoard[2])
		}
		return
	}

	if this.currProcCmd == "maps" || this.currProcCmd == "status" {
		//this.say( line)
		if this.multiLineRsltCmdComplete(cmdBody) {
			this.currProcCmd = ""
		}
		return
	}

	if cmdBody == "Banned players [ID]:" {
		if len(this.tmpBanList.BanList) > 0 {
			this.tmpBanList.BanList = this.tmpBanList.BanList[0:0]
			log.Printf("BanList init\n")
		}
		go func() {
			timer := time.NewTimer(time.Duration(60) * time.Second)
			<-timer.C
			this.updateBans()
		}()
		return
	}
	banIpId := BAN_REG_IP_ID.FindStringSubmatch(cmdBody)
	if banIpId != nil {
		this.tmpBanList.BanList = append(this.tmpBanList.BanList, Ban{banIpId[2], banIpId[3], banIpId[1], "", getNowTime()})
		log.Printf("BanIpId:%s,%s,%s\n", banIpId[2], banIpId[3], banIpId[1])
		return
	}
	banId := BAN_REG_ID.FindStringSubmatch(cmdBody)
	if banId != nil {
		this.tmpBanList.BanList = append(this.tmpBanList.BanList, Ban{banId[2], banId[1], "", "", ""})
		log.Printf("BanId:%s,%s\n", banId[1], banId[2])
		return
	}

	banIp := BAN_REG_IP.FindStringSubmatch(cmdBody)
	if banIp != nil {
		this.tmpBanList.BanList = append(this.tmpBanList.BanList, Ban{"", "", banIp[1], "", ""})
		log.Printf("BanIp:%s\n", banIp[1])
		return
	}
	banUser := BAN_REG_BAN.FindStringSubmatch(cmdBody)
	if banUser != nil {
		this.banUser(banUser[1], banUser[2])
		return
	}
	unbanUser := BAN_REG_UNBAN.FindStringSubmatch(cmdBody)
	if unbanUser != nil {
		this.unbanUser(unbanUser[1])
		return
	}

	if index > -1 && !strings.HasPrefix(cmdBody, "Server:") {
		userName := strings.TrimSpace(cmdBody[:index])
		uuid := this.getUserId(userName)
		if uuid == "" {
			//log.Printf("%s invalid user,vote invalid\n", userName)
			return
		}
		if _, ok := this.users[uuid]; ok {
			if userName == "Server" {
				return
			}
			sayBody := strings.TrimSpace(cmdBody[index+1:])
			if strings.HasPrefix(sayBody, "\\") || strings.HasPrefix(sayBody, "/") || strings.HasPrefix(sayBody, "!") {
				this.procUsrCmd(uuid, "", sayBody[1:])
			} else if len(this.votetickUsers) > 0 {
				if sayBody == "1" {
					log.Printf("%s votetick agree\n", userName)
					this.votetickUsers[uuid] = 1
				} else if sayBody == "0" {
					log.Printf("%s votetick not agree\n", userName)
					this.votetickUsers[uuid] = 0
				}
			} else {
				//fmt.Printf("%s : %s\n", userName, sayBody)
			}
		}
	}

	if strings.HasSuffix(cmdBody, "]") && strings.Index(cmdBody, USER_CONNECTED_KEY) > -1 {
		this.timeoutCnt = 0
		userName, uuid, isSucc := getUserByOutput(USER_CONNECTED_KEY, cmdBody)
		if !isSucc {
			log.Printf("[%s] invalid\n", cmdBody)
			return
		}
		lowName := strings.ToLower(userName)
		findIndex1 := strings.Index(lowName, "server")
		findIndex2 := strings.Index(lowName, "5erver")
		if findIndex1 > -1 || findIndex2 > -1 {
			this.say("error.login_forbbidden_username")
			this.execCmd("kick " + userName)
			return
		}
		this.onlineUser(userName, uuid)

		if this.users[uuid].IsAdmin {
			time.Sleep(1 * time.Second)
			if this.users[uuid].IsSuperAdmin {
				this.say("info.welcom_super_admin", userName)
			} else {
				this.say("info.welcom_admin", userName)
			}
			this.execCmd("admin add " + userName)
		}

	} else if strings.Index(cmdBody, USER_DISCONNECTED_KEY) > -1 {
		this.timeoutCnt = 0
		userName, uuid, isSucc := getUserByOutput(USER_DISCONNECTED_KEY, cmdBody)
		if !isSucc {
			log.Printf("[%s] invalid\n", cmdBody)
			return
		}
		if userName == "Server" {
			this.say("error.login_forbbidden_username")
			this.execCmd("kick " + userName)
			return
		}
		this.offlineUser(userName, uuid)
	} else if strings.Index(cmdBody, SERVER_READY_KEY) > -1 {
		this.offlineAllUser()
		this.serverIsRun = true

		this.execCmd("config name " + this.name)
		this.execCmd("config port " + strconv.Itoa(this.port))
		this.execCmd("bans")
		if this.mode == "mission" {
			this.execCmd("host " + this.missionMap + " mission")
		} else {
			this.execCmd("host Fortress")
		}
	} else if strings.Index(cmdBody, SERVER_STSRT_KEY) > -1 {
		log.Printf("server starting!\n")
		if this.firstIsStart {
			this.serverIsStart = true
			this.firstIsStart = false
			this.gameStatus.Running = "true"
		}
		this.serverIsRun = true
		this.offlineAllUser()
	}
}
func (this *Mindustry) run() {
	for {
		replaceJarReg, _ := regexp.Compile("\\S+\\.jar")
		javaParas := this.jarPath
		if this.mindustryVersionInfo.CurrVer != "" && this.mindustryVersionInfo.LocalFileName != "" {
			javaParas = replaceJarReg.ReplaceAllString(this.jarPath, this.mindustryVersionInfo.LocalFileName)
			this.currMindustryVer = this.mindustryVersionInfo.CurrVer
		}
		inPara := strings.Split(javaParas, " ")
		para := []string{"-jar"}
		index := strings.Index(javaParas, "-jar")
		if index < 0 {
			para = append(para, inPara...)
		} else {
			para = inPara
		}
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

func (this *Mindustry) startMapUpServer() {
	go func() {
		StartFileUpServer(this)
	}()
}
func main() {
	mode := flag.String("mode", "", "fix mode:survival,attack,sandbox,pvp")
	port := flag.Int("port", 0, "Input port")
	map_port := flag.Int("up", 0, "map up port")
	passwd := flag.String("gen", "", "crypto passwd")
	flag.Parse()
	if *passwd != "" {
		fmt.Println(cryptoMd5(*passwd))
		return
	}
	outfile, err := os.OpenFile("./logs/admin.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("open log file failed")
	} else {
		w := io.MultiWriter(os.Stdout, outfile)
		log.SetOutput(w)
	}
	log.Printf("version:%s!\n", _VERSION_)

	mindustry := Mindustry{}
	mindustry.init()
	if *port != 0 {
		mindustry.port = *port
	}
	if *map_port != 0 {
		mindustry.mapMangePort = *map_port
	}
	if *mode != "" {
		mindustry.mode = *mode
	}
	if mindustry.mode != "mission" {
		mindustry.startMapUpServer()
	}
	mindustry.run()
}
