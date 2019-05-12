package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/larspensjo/config"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func StripColor(str string) string {
	return re.ReplaceAllString(str, "")
}

type CallBack interface {
	output(line string, in io.WriteCloser)
}

func execCommand(commandName string, params []string, handle CallBack) error {
	cmd := exec.Command(commandName, params...)
	fmt.Println(cmd.Args)
	stdout, outErr := cmd.StdoutPipe()
	stdin, inErr := cmd.StdinPipe()
	//cmd.Stdin = os.Stdin
	if outErr != nil {
		return outErr
	}

	if inErr != nil {
		return inErr
	}
	cmd.Start()
	go func(cmd *exec.Cmd) {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGUSR1, syscall.SIGUSR2)
		s := <-c
		if cmd.Process != nil {
			log.Printf("sub process exit:%s", s)
			cmd.Process.Kill()
		}
	}(cmd)

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
	name          string
	isAdmin       bool
	isSupperAdmin bool
	level         int
}
type Cmd struct {
	name  string
	level int
}

type Mindustry struct {
	name       string
	admins     []string
	jarPath    string
	users      map[string]User
	serverOutR *regexp.Regexp
	cmds       map[string]Cmd
	port       int
	mode       string
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
				log.Printf("[ini]found supAdmins:%v\n", supAdmins)
				for _, supAdmin := range supAdmins {
					this.addUser(supAdmin)
					this.addSuperAdmin(supAdmin)
				}
			}
			optionValue, err = cfg.String("server", "adminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found adminCmds:%v\n", cmds)
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 1}
				}
			}
			optionValue, err = cfg.String("server", "superAdminCmds")
			if err == nil {
				optionValue := strings.TrimSpace(optionValue)
				cmds := strings.Split(optionValue, ",")
				log.Printf("[ini]found superAdminCmds:%v\n", cmds)
				for _, cmd := range cmds {
					this.cmds[cmd] = Cmd{cmd, 9}
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
}
func (this *Mindustry) init() {
	this.serverOutR, _ = regexp.Compile(".*(\\[INFO\\]|\\[ERR\\])(.*)")
	this.users = make(map[string]User)
	this.cmds = make(map[string]Cmd)
	rand.Seed(time.Now().UnixNano())
	this.name = fmt.Sprintf("mindustry-%d", rand.Int())
	this.jarPath = "server-release.jar"
	this.loadConfig()
	this.addUser("Server")
	this.addSuperAdmin("Server")

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
	tempUser.isSupperAdmin = true
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

	if !(this.users[name].isAdmin || this.users[name].isSupperAdmin) {
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
func (this *Mindustry) procUsrCmd(in io.WriteCloser, userName string, userInput string) {
	temps := strings.Split(userInput, " ")
	cmdName := temps[0]

	if cmd, ok := this.cmds[cmdName]; ok {
		if this.users[userName].level < cmd.level {
			info := fmt.Sprintf("user[%s] cmd :%s ,Permission denied!", userName, cmdName)
			say(in, info)
			return
		} else {
			info := fmt.Sprintf("proc user[%s] cmd :%s", userName, cmdName)
			say(in, info)
			if strings.EqualFold(userInput, "help") {
				say(in, "support cmds:")
				say(in, "admin cmds:")
			} else if strings.EqualFold(userInput, "gameover") {
				execCmd(in, "reloadmaps")
				execCmd(in, userInput)
			} else {
				execCmd(in, userInput)
			}
		}

	} else {
		info := fmt.Sprintf("proc user[%s] cmd :%s invalid!", userName, cmdName)
		say(in, info)
	}
}

const USER_CONNECTED_KEY string = " has connected."
const USER_DISCONNECTED_KEY string = " has disconnected."
const SERVER_INFO_LOG string = "[INFO] "
const SERVER_READY_KEY string = "Server loaded. Type 'help' for help."

func (this *Mindustry) output(line string, in io.WriteCloser) {

	index := strings.Index(line, SERVER_INFO_LOG)
	if index < 0 {
		return
	}

	cmdBody := strings.TrimSpace(line[index+len(SERVER_INFO_LOG):])
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
				fmt.Printf("%s : %s\n", userName, sayBody)
			}
		}
	}

	if strings.HasSuffix(cmdBody, USER_CONNECTED_KEY) {
		userName := strings.TrimSpace(cmdBody[:len(cmdBody)-len(USER_CONNECTED_KEY)])
		this.onlineUser(userName)

		if this.users[userName].isAdmin {
			time.Sleep(1 * time.Second)
			say(in, "Welcome admin:"+userName)
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

	startMapUpServer(*map_port)
	mindustry := Mindustry{}
	mindustry.init()
	mindustry.mode = *mode
	mindustry.port = *port
	mindustry.run()
}
