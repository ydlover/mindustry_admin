package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/larspensjo/config"
	"github.com/robfig/cron"
)

var _VERSION_ = "1.0"

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripColor(str string) string {
	return re.ReplaceAllString(str, "")
}

type CallBack interface {
	output(line string, in io.WriteCloser)
}
type UserCmdProcHandle func(in io.WriteCloser, userName string, userInput string)

func execCommand(commandName string, params []string, handle CallBack) error {
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
		hour := time.Now().Hour()
		execCmd(stdin, "save "+strconv.Itoa(hour))
		say(stdin, "auto save "+strconv.Itoa(hour))
	})
	c.Start()
	go func(cmd *exec.Cmd) {
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			execCmd(stdin, strings.TrimRight(line, "\n"))
		}
	}(cmd)

	reader := bufio.NewReader(stdout)

	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Printf(line)
		handle.output(StripColor(line), stdin)
	}
	cmd.Wait()
	return nil
}

type User struct {
	name         string
	isAdmin      bool
	isSuperAdmin bool
	level        int
}
type Cmd struct {
	name  string
	level int
}

type Mindustry struct {
	name               string
	admins             []string
	cfgAdmin           string
	cfgSuperAdmin      string
	jarPath            string
	users              map[string]User
	serverOutR         *regexp.Regexp
	cfgAdminCmds       string
	cfgSuperAdminCmds  string
	cfgNormCmds        string
	cmds               map[string]Cmd
	cmdHelps           map[string]string
	port               int
	mode               string
	cmdFailReason      string
	currProcCmd        string
	playCnt            int
	serverIsRun        bool
	maps               []string
	userCmdProcHandles map[string]UserCmdProcHandle
}

func (this *Mindustry) loadConfig() {

	cfg, err := config.ReadDefault("config.ini")
	if err != nil {
		log.Println("[ini]not find config.ini,use default config")
		return
	}
	if cfg.HasSection("server") {
		_, err := cfg.SectionOptions("server")
		if err == nil {
			optionValue := ""
			optionValue, err = cfg.String("server", "admins")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				admins := strings.Split(optionValue, ",")
				this.cfgAdmin = optionValue
				log.Printf("[ini]found admins:%v\n", admins)
				for _, admin := range admins {
					this.addUser(admin)
					this.addAdmin(admin)
				}
			}
			optionValue, err = cfg.String("server", "superAdmins")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				supAdmins := strings.Split(optionValue, ",")
				this.cfgSuperAdmin = optionValue
				log.Printf("[ini]found supAdmins:%v\n", supAdmins)
				for _, supAdmin := range supAdmins {
					this.addUser(supAdmin)
					this.addSuperAdmin(supAdmin)
				}
			}
			optionValue, err = cfg.String("server", "superAdminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				this.cfgSuperAdminCmds = optionValue
				log.Printf("[ini]found superAdminCmds:%v\n", cmds)
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 9}
				}
			}

			optionValue, err = cfg.String("server", "adminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found adminCmds:%v\n", cmds)
				this.cfgAdminCmds = optionValue
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 1}
				}
			}
			optionValue, err = cfg.String("server", "normCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found normCmds:%v\n", cmds)
				this.cfgNormCmds = optionValue
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 0}
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
		}
	}

	if cfg.HasSection("cmdHelps") {
		section, err := cfg.SectionOptions("cmdHelps")
		if err == nil {
			for _, v := range section {
				options, err := cfg.String("cmdHelps", v)
				if err == nil {
					//log.Printf("[ini]found help:%s %s\n", v, options)
					this.cmdHelps[v] = options
				}
			}
		}
	}
}
func (this *Mindustry) init() {
	this.serverOutR, _ = regexp.Compile(".*(\\[INFO\\]|\\[ERR\\])(.*)")
	this.users = make(map[string]User)
	this.cmds = make(map[string]Cmd)
	this.cmdHelps = make(map[string]string)
	this.userCmdProcHandles = make(map[string]UserCmdProcHandle)
	rand.Seed(time.Now().UnixNano())
	this.name = fmt.Sprintf("mindustry-%d", rand.Int())
	this.jarPath = "server-release.jar"
	this.loadConfig()
	this.addUser("Server")
	this.addSuperAdmin("Server")
	this.userCmdProcHandles["admin"] = this.proc_admin
	this.userCmdProcHandles["directCmd"] = this.proc_directCmd
	this.userCmdProcHandles["gameover"] = this.proc_gameover
	this.userCmdProcHandles["help"] = this.proc_help
	this.userCmdProcHandles["host"] = this.proc_host
	this.userCmdProcHandles["hostx"] = this.proc_host
	this.userCmdProcHandles["save"] = this.proc_save
	this.userCmdProcHandles["load"] = this.proc_load
	this.userCmdProcHandles["maps"] = this.proc_mapsOrStatus
	this.userCmdProcHandles["status"] = this.proc_mapsOrStatus
	this.userCmdProcHandles["slots"] = this.proc_slots
	this.userCmdProcHandles["showAdmin"] = this.proc_showAdmin
	this.userCmdProcHandles["show"] = this.proc_show

}
func (this *Mindustry) addUser(name string) {
	if _, ok := this.users[name]; ok {
		return
	}
	this.users[name] = User{name, false, false, 0}
	log.Printf("add user info :%s\n", name)
}
func (this *Mindustry) addAdmin(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("user %s not found\n", name)
		return
	}
	tempUser := this.users[name]
	tempUser.isAdmin = true
	tempUser.level = 1
	this.users[name] = tempUser
	log.Printf("add admin :%s\n", name)
}

func (this *Mindustry) addSuperAdmin(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("user %s not found\n", name)
		return
	}
	tempUser := this.users[name]
	tempUser.isAdmin = true
	tempUser.isSuperAdmin = true
	tempUser.level = 9
	this.users[name] = tempUser
	log.Printf("add superAdmin :%s\n", name)
}

func (this *Mindustry) onlineUser(name string) {
	if _, ok := this.users[name]; ok {
		return
	}
	this.addUser(name)
}
func (this *Mindustry) offlineUser(name string) {
	if _, ok := this.users[name]; ok {
		return
	}

	if !(this.users[name].isAdmin || this.users[name].isSuperAdmin) {
		this.delUser(name)
		return
	}
}
func (this *Mindustry) delUser(name string) {
	if _, ok := this.users[name]; !ok {
		log.Printf("del user not exist :%s\n", name)
		return
	}
	delete(this.users, name)
	log.Printf("del user info :%s\n", name)
}
func execCmd(in io.WriteCloser, cmd string) {

	log.Printf("execCmd :%s\n", cmd)
	data := []byte(cmd + "\n")
	in.Write(data)
}

func say(in io.WriteCloser, cmd string) {
	data := []byte("say " + cmd + "\n")
	in.Write(data)
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

func (this *Mindustry) proc_mapsOrStatus(in io.WriteCloser, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	cmdName := temps[0]

	if cmdName == "maps" || cmdName == "status" {
		go func() {
			timer := time.NewTimer(time.Duration(5) * time.Second)
			<-timer.C
			if this.currProcCmd != "" {
				say(in, "Command "+this.currProcCmd+" timeout!")
				this.currProcCmd = ""
			}
		}()
		this.currProcCmd = cmdName
	}
	if cmdName == "maps" {
		execCmd(in, "reloadmaps")
		this.maps = this.maps[0:0]
		execCmd(in, "maps")
	} else if cmdName == "status" {
		execCmd(in, "status")
	}
}
func (this *Mindustry) proc_host(in io.WriteCloser, userName string, userInput string) {
	mapName := ""
	temps := strings.Split(userInput, " ")
	if len(temps) < 2 {
		say(in, "Command ("+userInput+") length invalid!")
		return
	}
	inputCmd := strings.TrimSpace(temps[0])
	inputMap := strings.TrimSpace(temps[1])
	inputMode := ""
	if len(temps) > 2 {
		inputMode = strings.TrimSpace(temps[2])
	}
	if inputCmd == "hostx" {
		inputIndex := 0
		var err error = nil
		if inputIndex, err = strconv.Atoi(inputMap); err != nil {
			say(in, "Command ("+userInput+") invalid,please input number!")
			return
		}
		if inputIndex < 0 || inputIndex >= len(this.maps) {

			say(in, "Command ("+userInput+") invalid,mapIndex err!")
			return
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
			say(in, "Command ("+userInput+") invalid,map not found!")
			return
		}
	} else {
		say(in, "Command ("+userInput+") invalid!")
		return
	}
	if inputMode != "pvp" && inputMode != "attack" && inputMode != "" && inputMode != "sandbox" {
		say(in, "Command ("+userInput+") invalid,mode  err!")
		return
	}
	say(in, "The server needs to be restarted. Please wait 10 seconds to log in!")
	execCmd(in, "reloadmaps")
	time.Sleep(time.Duration(5) * time.Second)
	execCmd(in, "stop")
	time.Sleep(time.Duration(5) * time.Second)
	if inputMode == "" {
		execCmd(in, "host "+mapName)
	} else {
		execCmd(in, "host "+mapName+" "+inputMode)
	}
}

func (this *Mindustry) proc_save(in io.WriteCloser, userName string, userInput string) {
	targetSlot := ""
	if userInput == "save" {
		minute := time.Now().Minute()
		targetSlot = fmt.Sprintf("%d%02d%02d", time.Now().Day(), time.Now().Hour(), minute/10*10)
	} else {
		targetSlot = userInput[len("save"):]
		targetSlot = strings.TrimSpace(targetSlot)
	}
	if _, ok := strconv.Atoi(targetSlot); ok != nil {
		say(in, "slot invalid:"+targetSlot+",please input number,ie:save 111")
		return
	}
	execCmd(in, "save "+targetSlot)
	say(in, "save slot("+targetSlot+") success!")
}

func (this *Mindustry) proc_load(in io.WriteCloser, userName string, userInput string) {
	targetSlot := userInput[len("load"):]
	targetSlot = strings.TrimSpace(targetSlot)
	if !checkSlotValid(targetSlot) {
		say(in, "load slot not exist,please check input:"+targetSlot)
		return
	}
	say(in, "The server needs to be restarted. Please wait 10 seconds to log in.!")
	time.Sleep(time.Duration(5) * time.Second)
	execCmd(in, "stop")
	time.Sleep(time.Duration(5) * time.Second)
	execCmd(in, userInput)
}
func (this *Mindustry) proc_admin(in io.WriteCloser, userName string, userInput string) {
	targetName := userInput[len("admin"):]
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		say(in, "Please input admin name")
	} else {
		this.addAdmin(targetName)
		execCmd(in, userInput)
		say(in, "admin ["+targetName+"] is add!")
	}
}
func (this *Mindustry) proc_directCmd(in io.WriteCloser, userName string, userInput string) {
	execCmd(in, userInput)
}
func (this *Mindustry) proc_gameover(in io.WriteCloser, userName string, userInput string) {
	execCmd(in, "reloadmaps")
	execCmd(in, userInput)
}
func (this *Mindustry) proc_help(in io.WriteCloser, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	if len(temps) >= 2 {
		cmd := strings.TrimSpace(temps[1])
		if helpInfo, ok := this.cmdHelps[cmd]; ok {
			say(in, cmd+" "+helpInfo)
		} else {
			say(in, "invalid cmd:"+cmd)
		}
	} else {
		if this.users[userName].isSuperAdmin {
			say(in, "super admin cmd:"+this.cfgSuperAdminCmds)
		} else if this.users[userName].isAdmin {
			say(in, "admin cmd:"+this.cfgAdminCmds)
		} else {
			say(in, "user cmd:"+this.cfgNormCmds)
		}

	}
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
func (this *Mindustry) proc_show(in io.WriteCloser, userName string, userInput string) {
	say(in, "Ver:"+_VERSION_)
	tempStr := fmt.Sprintf("CPU temperature: %.3f°C", getCpuTemp())
	say(in, tempStr)
}
func (this *Mindustry) proc_showAdmin(in io.WriteCloser, userName string, userInput string) {
	say(in, "super admin:"+this.cfgSuperAdmin)
	say(in, "admin:"+this.cfgAdmin)

}

func (this *Mindustry) proc_slots(in io.WriteCloser, userName string, userInput string) {
	say(in, "slots:"+getSlotList())
}
func (this *Mindustry) procUsrCmd(in io.WriteCloser, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	cmdName := temps[0]

	if cmd, ok := this.cmds[cmdName]; ok {
		if this.users[userName].level < cmd.level {
			info := fmt.Sprintf("user[%s] cmd :%s ,Permission denied!", userName, cmdName)
			say(in, info)
			return
		} else {
			if this.currProcCmd != "" {
				say(in, "Command "+this.currProcCmd+" is executing, please wait for execution to complete!")
				return
			}

			//info := fmt.Sprintf("proc user[%s] cmd :%s", userName, userInput)
			//say(in, info)
			if handleFunc, ok := this.userCmdProcHandles[cmdName]; ok {
				handleFunc(in, userName, userInput)
			} else {
				this.userCmdProcHandles["directCmd"](in, userName, userInput)
			}
		}

	} else {
		info := fmt.Sprintf("proc user[%s] cmd :%s invalid!", userName, cmdName)
		say(in, info)
	}
}
func (this *Mindustry) multiLineRsltCmdComplete(in io.WriteCloser, line string) bool {
	index := -1
	if this.currProcCmd == "maps" {
		if strings.Index(line, "Map directory:") >= 0 {
			mapsInfo := "maps:"
			for index, name := range this.maps {
				if mapsInfo != "maps:" {
					mapsInfo += ","
				}
				mapsInfo += ("[" + strconv.Itoa(index) + "]" + name)
			}
			say(in, mapsInfo)
			return true
		}
		mapNameEndIndex := -1
		index = strings.Index(line, ": Custom /")
		if index >= 0 {
			mapNameEndIndex = index
		}
		index = strings.Index(line, ": Default /")
		if index >= 0 {
			mapNameEndIndex = index
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

func (this *Mindustry) output(line string, in io.WriteCloser) {

	index := strings.Index(line, SERVER_ERR_LOG)
	if index >= 0 {
		errInfo := strings.TrimSpace(line[index+len(SERVER_ERR_LOG):])
		if strings.Contains(errInfo, "io.anuke.arc.util.ArcRuntimeException: File not found") {
			log.Printf("map not found , force exit!\n")
			execCmd(in, "exit")
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
		//say(in, line)
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
			if strings.HasPrefix(sayBody, "\\") {
				this.procUsrCmd(in, userName, sayBody[1:])
			} else {
				//fmt.Printf("%s : %s\n", userName, sayBody)
			}
		}
	}

	if strings.HasSuffix(cmdBody, USER_CONNECTED_KEY) {
		userName := strings.TrimSpace(cmdBody[:len(cmdBody)-len(USER_CONNECTED_KEY)])
		this.onlineUser(userName)

		if this.users[userName].isAdmin {
			time.Sleep(1 * time.Second)
			if this.users[userName].isSuperAdmin {
				say(in, "Welcome Super admin:::::::::::::::: "+userName)
			} else {
				say(in, "Welcome admin:"+userName)
			}
			execCmd(in, "admin "+userName)
		}

	} else if strings.HasSuffix(cmdBody, USER_DISCONNECTED_KEY) {
		userName := strings.TrimSpace(cmdBody[:len(cmdBody)-len(USER_DISCONNECTED_KEY)])
		this.offlineUser(userName)
	} else if strings.HasPrefix(cmdBody, SERVER_READY_KEY) {
		execCmd(in, "port "+strconv.Itoa(this.port))
		if this.mode == "mission" {
			execCmd(in, "host 8 "+this.mode)
		} else {
			this.serverIsRun = true
			execCmd(in, "host Fortress "+this.mode)
		}
	} else {

	}
}
func (this *Mindustry) run() {
	var para = []string{"-jar", this.jarPath}
	for {
		execCommand("java", para, this)
		log.Printf("server crash,wait(10s) reboot!\n")
		time.Sleep(time.Duration(10) * time.Second)
	}
}
func startMapUpServer(port int) {
	go func(serverPort int) {
		StartFileUpServer(serverPort)
	}(port)
}
func main() {
	mode := flag.String("mode", "survival", "mode:survival,attack,sandbox,pvp,mission")
	port := flag.Int("port", 6567, "Input port")
	map_port := flag.Int("up", 6569, "map up port")
	flag.Parse()
	log.Printf("version:%s!\n", _VERSION_)

	startMapUpServer(*map_port)
	mindustry := Mindustry{}
	mindustry.init()
	mindustry.mode = *mode
	mindustry.port = *port
	mindustry.run()
}
