package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var url2path = map[string]string{
	"/maps":    "./config/maps/",
	"/mods":    "./config/mods/",
	"/plugins": "./config/plugins/",
}

type FileDesc struct {
	Id   int    `json:"id"`
	Size int64  `json:"size"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type AdminsList struct {
	UserName  string `json:"username"`
	GameName  string `json:"gamename"`
	ApplyTime string `json:"time"`
}
type LoginRslt struct {
	Result  string `json:"result"`
	Session string `json:"session"`
}

type Rslt struct {
	Result string `json:"result"`
}

func DirsInit() {
	for _, filePath := range url2path {
		err := initFilePath(filePath)
		if err != nil {
			fmt.Println(err)
		}
	}
}

const RSP_SUCC string = "succ"
const RSP_FAIL string = "fail"

var m_mindustryServer *Mindustry

func StartFileUpServer(mindustryServer *Mindustry) {
	DirsInit()
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("web"))
	mux.Handle("/", fs)
	mh := http.HandlerFunc(handleFilesRequest)
	mux.Handle("/login", http.HandlerFunc(handleLoginRequest))
	mux.Handle("/sign", http.HandlerFunc(handleSignRequest))
	mux.Handle("/signList", http.HandlerFunc(handleSignList))
	mux.Handle("/modifyPasswd", http.HandlerFunc(handleModifyPasswdRequest))
	mux.Handle("/resetUuid", http.HandlerFunc(handleResetUuidRequest))
	mux.Handle("/admins", http.HandlerFunc(handleAdminsRequest))
	mux.Handle("/blacklist", http.HandlerFunc(handleBlackListRequest))
	mux.Handle("/status", http.HandlerFunc(handleStatusRequest))
	mux.Handle("/maintain", http.HandlerFunc(handleMaintainRequest))

	for url, _ := range url2path {
		mux.Handle(url, mh)
	}
	server := &http.Server{
		Addr:    "0.0.0.0:" + strconv.Itoa(mindustryServer.mapMangePort),
		Handler: mux,
	}
	fmt.Println("file up server listening on: http://0.0.0.0:" + strconv.Itoa(mindustryServer.mapMangePort))
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill)
		s := <-c
		server.Close()
		fmt.Printf("file up server shutdownf%s", s)
	}()
	m_mindustryServer = mindustryServer
	server.ListenAndServe()
}

func authRequest(w http.ResponseWriter, userName string, sessionId string) bool {
	fmt.Printf("auth:username=%s,sessionId=%s\n", userName, sessionId)
	if m_mindustryServer.webLoginSessionChk(userName, sessionId) {
		return true
	}
	var result Rslt
	result.Result = "user not login!"
	output, err1 := json.MarshalIndent(&result, "", "\t\t")
	if err1 != nil {
		fmt.Printf("json gen fail")
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
	return false
}

func Response(w http.ResponseWriter, r string) {
	var result Rslt
	result.Result = r
	output, err1 := json.MarshalIndent(&result, "", "\t\t")
	if err1 != nil {
		fmt.Printf("json gen fail")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func handleLoginRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("handleLoginRequest url:" + r.URL.Path + ", method:" + r.Method)

	switch r.Method {
	case "POST":

		r.ParseForm()
		userName := r.Form.Get("username")
		passwd := r.Form.Get("passwd")
		fmt.Println("request name:" + userName)
		fmt.Println("request pass:" + passwd)
		var result LoginRslt
		isSucc := m_mindustryServer.webLoginAdmin(userName, passwd)
		if isSucc {
			result.Result = "admin"
			if m_mindustryServer.webLoginIsSop(userName) {
				result.Result = "suop"
			}
			result.Session = m_mindustryServer.getAdmin(userName).sessionId
		} else {
			result.Result = RSP_FAIL
		}
		output, err1 := json.MarshalIndent(&result, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
func handleSignRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleSignRequest url:" + r.URL.Path + ", method:" + r.Method)
	switch r.Method {
	case "POST":
		r.ParseForm()
		userName := r.Form.Get("username")
		gameName := r.Form.Get("gamename")
		passwd := r.Form.Get("passwd")
		contact := r.Form.Get("contact")
		isSucc := m_mindustryServer.regAdmin(userName, gameName, passwd, contact)
		if isSucc {
			Response(w, RSP_SUCC)
		} else {
			Response(w, RSP_FAIL)
		}

	}
}
func handleResetUuidRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleResetUuidRequest url:" + r.URL.Path + ", method:" + r.Method)
	switch r.Method {
	case "GET":
		query := r.URL.Query()
		userName := query.Get("username")
		gameName := query.Get("gamename")
		sessionId := query.Get("sessionid")
		if !authRequest(w, userName, sessionId) {
			return
		}
		if gameName == "" {
			gameName = m_mindustryServer.getAdmin(userName).Name
		}
		isSucc := m_mindustryServer.webLoginUuidReset(userName, gameName)
		if isSucc {
			Response(w, RSP_SUCC)
		} else {
			Response(w, RSP_FAIL)
		}

	}
}

func handleModifyPasswdRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleModifyPasswdRequest url:" + r.URL.Path + ", method:" + r.Method)
	switch r.Method {
	case "POST":
		r.ParseForm()
		if !authRequest(w, r.Form.Get("username"), r.Form.Get("sessionid")) {
			return
		}
		userName := r.Form.Get("username")
		passwd := r.Form.Get("passwd")

		isSucc := m_mindustryServer.webLoginModifyPasswd(userName, passwd)
		if isSucc {
			Response(w, RSP_SUCC)
		} else {
			Response(w, RSP_FAIL)
		}

		return
	}
}
func handleBlackListRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleBlackListRequest url:" + r.URL.Path + ", method:" + r.Method)
	switch r.Method {
	case "GET":
		query := r.URL.Query()
		userName := query.Get("username")
		if !authRequest(w, userName, query.Get("sessionid")) {
			return
		}

		unbanTarget := query.Get("unban")
		if unbanTarget != "" {
			fmt.Println("(" + userName + ")Req unban:" + unbanTarget)
			m_mindustryServer.execCmd("unban " + unbanTarget)
			Response(w, RSP_SUCC)
			return
		}

		output, err1 := json.MarshalIndent(m_mindustryServer.currBanList.BanList, "", "\t\t")
		if err1 != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	case "POST":
		Response(w, RSP_FAIL)
	}
}

func handleSignList(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleSignList url:" + r.URL.Path)

	switch r.Method {
	case "GET":
		query := r.URL.Query()
		userName := query.Get("username")
		sessionId := query.Get("sessionid")
		if !authRequest(w, userName, sessionId) {
			return
		}
		deny := query.Get("deny")
		add := query.Get("add")
		isSop := m_mindustryServer.webLoginIsSop(userName)
		if (deny != "" || add != "") && !isSop {
			Response(w, "user not is super admin")
			return
		}
		if deny != "" {
			isExist := m_mindustryServer.getSign(deny) != nil
			if !isExist {
				Response(w, "deny user is not exist")
				return
			}
			denyIsSucc := m_mindustryServer.denySign(deny)
			if !denyIsSucc {
				Response(w, "user deny fail")
				return
			}
			Response(w, RSP_SUCC)
			return
		}

		if add != "" {
			isExist := m_mindustryServer.getAdmin(add) != nil
			if isExist {
				Response(w, "user is exist")
				return
			}
			if !m_mindustryServer.addAdmin(add) {
				Response(w, "user add fail")
				return
			}
			Response(w, RSP_SUCC)
			return
		}

		output, err1 := json.MarshalIndent(m_mindustryServer.adminCfg.SignList, "", "\t\t")
		if err1 != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	case "POST":
		Response(w, RSP_FAIL)
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func handleMaintainRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		query := r.URL.Query()
		userName := query.Get("username")
		sessionId := query.Get("sessionid")
		if !authRequest(w, userName, sessionId) {
			return
		}
		say := query.Get("say")
		reboot := query.Get("reboot")
		isSop := m_mindustryServer.webLoginIsSop(userName)
		if (reboot != "") && !isSop {
			Response(w, "user not is super admin")
			return
		}
		if say != "" {
			maxLen := min(len(say), 100)
			say = "(" + userName + ") " + say[0:maxLen-1]
			m_mindustryServer.say(say)
			Response(w, RSP_SUCC)
			return
		}

		messageQuery := query.Get("chatMessage")
		if messageQuery != "" {
			ret := make([]Message, 0)
			begin, error := strconv.Atoi(messageQuery)
			if error == nil {
				ret = m_mindustryServer.getChartMesaage(begin)
			} else {
				fmt.Println("chatMessage para invalid:" + messageQuery)
			}
			output, err1 := json.MarshalIndent(ret, "", "\t\t")
			if err1 != nil {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(output)
			return
		}
		userQuery := query.Get("users")
		if userQuery != "" {
			ret := make([]User, 0)
			isOnline, error := strconv.Atoi(userQuery)
			if error == nil {
				ret = m_mindustryServer.getUserListForWeb(isOnline == 1)
			} else {
				fmt.Println("userQuery para invalid:" + userQuery)
			}
			output, err1 := json.MarshalIndent(ret, "", "\t\t")
			if err1 != nil {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(output)
			return
		}

		banTarget := query.Get("ban")
		if banTarget != "" {
			fmt.Println("(" + userName + ")Req ban:" + banTarget)
			m_mindustryServer.execCmd("ban id " + banTarget)
			Response(w, RSP_SUCC)
			return
		}

	}
}

func handleAdminsRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleAdminsRequest url:" + r.URL.Path)
	switch r.Method {
	case "GET":
		query := r.URL.Query()
		userName := query.Get("username")
		sessionId := query.Get("sessionid")
		if !authRequest(w, userName, sessionId) {
			return
		}
		rmv := query.Get("rmv")
		isSop := m_mindustryServer.webLoginIsSop(userName)
		if rmv != "" && !isSop {
			Response(w, "user not is super admin")
			return
		}
		if rmv != "" {
			isExist := m_mindustryServer.getAdmin(rmv) != nil
			if !isExist {
				Response(w, "user is not exist")
				return
			}
			if !m_mindustryServer.rmvAdmin(rmv) {
				Response(w, "user rmv fail")
				return
			}
			Response(w, RSP_SUCC)
			return
		}
		adminList := make([]Admin, len(m_mindustryServer.adminCfg.AdminList))
		copy(adminList, m_mindustryServer.adminCfg.AdminList)
		for i, _ := range adminList {
			if adminList[i].Id != "" {
				adminList[i].Id = "true"
			} else {
				adminList[i].Id = "false"
			}
		}
		output, err1 := json.MarshalIndent(adminList, "", "\t\t")
		if err1 != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	case "POST":
		Response(w, RSP_FAIL)
	}
}
func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleStatusRequest url:" + r.URL.Path)
	switch r.Method {
	case "GET":
		m_mindustryServer.updateStatus()
		output, err1 := json.MarshalIndent(m_mindustryServer.gameStatus, "", "\t\t")
		if err1 != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	}
}

func getRequestFileUrl(r *http.Request) string {
	for url, _ := range url2path {
		if strings.HasPrefix(r.URL.Path, url) {
			return url
		}
	}
	return ""
}

func getDirFilesCnt(requestUrl string) int {
	files, _ := ioutil.ReadDir(url2path[requestUrl])
	return len(files)
}
func handleFilesRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("request files url:" + r.URL.Path)
	requestUrl := getRequestFileUrl(r)
	if requestUrl == "" {
		Response(w, "invalid url")
		return
	}
	switch r.Method {
	case "GET":
		err = handleFilesGet(requestUrl, w, r)
	case "POST":
		fileCnt := getDirFilesCnt(requestUrl)
		if fileCnt > m_mindustryServer.maxMapCount {
			fmt.Printf("fileCnt(%d) > maxMapCount(%d)!\n", fileCnt, m_mindustryServer.maxMapCount)
			Response(w, "file is full")
			return
		}
		err = handlePost(requestUrl, w, r)
		if err != nil {
			return
		}
	}
}

func handleFilesGet(requestUrl string, w http.ResponseWriter, r *http.Request) (err error) {
	//fmt.Println("GET: " + r.URL.Path)
	query := r.URL.Query()
	if !authRequest(w, query.Get("username"), query.Get("sessionid")) {
		return
	}
	delFile := query.Get("delete")
	downFile := query.Get("download")
	if downFile != "" {
		fmt.Println("download: " + downFile)
		file := url2path[requestUrl] + downFile
		if exist, _ := exists(file); !exist {
			http.NotFound(w, r)
		}
		http.ServeFile(w, r, file)
		return
	}
	if delFile != "" {
		fmt.Println("DELETE: " + url2path[requestUrl] + delFile)
		err = os.Remove(url2path[requestUrl] + delFile)
		if err != nil {
			fmt.Println(err)
			Response(w, "file del fail")
		} else {
			Response(w, RSP_SUCC)
		}
		return
	}
	_dirpath, err1 := os.Open(url2path[requestUrl])
	if err1 != nil {
		err = err1
		return
	}
	_dir, err1 := _dirpath.Readdir(0)
	if err1 != nil {
		err = err1
		return
	}
	files := make([]FileDesc, len(_dir))
	for i, f := range _dir {
		files[i].Id = i + 1
		files[i].Name = f.Name()
		files[i].Path = ""
		files[i].Size = f.Size()

	}
	output, err1 := json.MarshalIndent(&files, "", "\t\t")
	if err1 != nil {
		err = err1
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
	return

}

func handlePost(requestUrl string, w http.ResponseWriter, r *http.Request) (err error) {
	fmt.Println("POST: " + r.URL.Path)
	r.ParseForm()
	userName := r.Form.Get("username")
	sessionId := r.Form.Get("sessionid")
	if !authRequest(w, userName, sessionId) {
		return
	}
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Fprintf(w, "%v", handler.Header)
	f, err := os.OpenFile(url2path[requestUrl]+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)
	return
}
func handleDelete(requestUrl string, w http.ResponseWriter, r *http.Request) (err error) {
	name := path.Base(r.URL.Path)
	fmt.Println("DELETE: " + r.URL.Path + "," + name)
	err = os.Remove(url2path[requestUrl] + name)
	if err != nil {
		fmt.Println(err)
	}
	w.WriteHeader(200)
	return
}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}
func initFilePath(filePath string) (err error) {
	isExist, _ := exists(filePath)
	if isExist {
		return
	}
	err = os.Mkdir(filePath, 0777)
	return
}
func exists(path string) (bool, error) {

	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}
func check(name string) bool {
	ext := []string{".msav"}

	for _, v := range ext {
		if v == name {
			return false
		}
	}
	return true
}
