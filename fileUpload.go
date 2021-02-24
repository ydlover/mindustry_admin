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

func init() {
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
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("map_manager"))
	mux.Handle("/", fs)
	mh := http.HandlerFunc(handleFilesRequest)
	mux.Handle("/login", http.HandlerFunc(handleLoginRequest))
	mux.Handle("/sign", http.HandlerFunc(handleSignRequest))
	mux.Handle("/modifyPasswd", http.HandlerFunc(handleModifyPasswdRequest))
	mux.Handle("/resetUuid", http.HandlerFunc(handleResetUuidRequest))
	mux.Handle("/admins", http.HandlerFunc(handleAdminsRequest))
	mux.Handle("/blacklist", http.HandlerFunc(handleBlackListRequest))

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

func authRequest(w http.ResponseWriter, username string, sessionId string) bool {
	fmt.Print("auth:username=%s,sessionId=%s\n", username, sessionId)
	/*
		var result Rslt
		result.Result = "user not login!"
		output, err1 := json.MarshalIndent(&result, "", "\t\t")
		if err1 != nil {
			fmt.Printf("json gen fail")
			return false
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
	*/
	return true
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
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))

	switch r.Method {
	case "POST":
		var result LoginRslt
		result.Result = "admin"
		result.Session = "123123123123"
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
	var err error
	fmt.Println("handleSignRequest url:" + r.URL.Path + ", method:" + r.Method)
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))

	switch r.Method {
	case "POST":
		var result Rslt
		result.Result = "succ"
		output, err1 := json.MarshalIndent(&result, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
func handleResetUuidRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("handleResetUuidRequest url:" + r.URL.Path + ", method:" + r.Method)
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))

	switch r.Method {
	case "GET":
		query := r.URL.Query()
		if !authRequest(w, query.Get("username"), query.Get("sessionid")) {
			return
		}
		var result Rslt
		result.Result = "succ"
		output, err1 := json.MarshalIndent(&result, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func handleModifyPasswdRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("handleModifyPasswdRequest url:" + r.URL.Path + ", method:" + r.Method)
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))
	switch r.Method {
	case "POST":
		r.ParseForm()
		if !authRequest(w, r.Form.Get("username"), r.Form.Get("sessionid")) {
			return
		}
		var result Rslt
		result.Result = "succ"
		output, err1 := json.MarshalIndent(&result, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
func handleBlackListRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("handleBlackListRequest url:" + r.URL.Path + ", method:" + r.Method)
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))

	switch r.Method {
	case "GET":
		query := r.URL.Query()
		if !authRequest(w, query.Get("username"), query.Get("sessionid")) {
			return
		}

		unbanTarget := query.Get("unban")
		if unbanTarget != "" {
			fmt.Println("Req unban:" + unbanTarget)
			m_mindustryServer.execCmd("unban " + unbanTarget)
			Response(w, RSP_SUCC)
			return
		}

		output, err1 := json.MarshalIndent(m_mindustryServer.currBanList.BanList, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	case "POST":

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
func handleAdminsRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	fmt.Println("handleAdminsRequest url:" + r.URL.Path)
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))

	switch r.Method {
	case "GET":
		query := r.URL.Query()
		if !authRequest(w, query.Get("username"), query.Get("sessionid")) {
			return
		}
		adminsList := make([]AdminsList, 3)
		adminsList[0].UserName = "user1"
		adminsList[0].GameName = "gameUser1"
		adminsList[0].ApplyTime = "2021-01-09 22:00:00"
		adminsList[1].UserName = "user1"
		adminsList[1].GameName = "gameUser1"
		adminsList[1].ApplyTime = "2021-01-09 22:00:00"
		adminsList[2].UserName = "user1"
		adminsList[2].GameName = "gameUser1"
		adminsList[2].ApplyTime = "2021-01-09 22:00:00"

		output, err1 := json.MarshalIndent(&adminsList, "", "\t\t")
		if err1 != nil {
			err = err1
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		return
	case "POST":

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
	con, _ := ioutil.ReadAll(r.Body) //获取post的数据
	fmt.Println("request post:" + string(con))
	requestUrl := getRequestFileUrl(r)
	if requestUrl == "" {
		fmt.Printf("invalid url!\n")
		w.WriteHeader(403)
		w.Write([]byte("not_permit"))
		return
	}
	switch r.Method {
	case "GET":
		err = handleFilesGet(requestUrl, w, r)
	case "POST":
		if !m_mindustryServer.isPermitMapModify() {
			fmt.Printf("up is not permit!\n")
			w.WriteHeader(403)
			w.Write([]byte("not_permit"))
			return
		}
		fileCnt := getDirFilesCnt(requestUrl)
		if fileCnt > m_mindustryServer.maxMapCount {
			fmt.Printf("fileCnt(%d) > maxMapCount(%d)!\n", fileCnt, m_mindustryServer.maxMapCount)
			w.WriteHeader(403)
			w.Write([]byte("not_cap"))
			return
		}
		err = handlePost(requestUrl, w, r)

	case "DELETE":
		if !m_mindustryServer.isPermitMapModify() {
			fmt.Printf("map delete is not permit!\n")
			w.WriteHeader(200)
			w.Write([]byte("not_permit"))
			return
		}
		err = handleDelete(requestUrl, w, r)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleFilesGet(requestUrl string, w http.ResponseWriter, r *http.Request) (err error) {
	//fmt.Println("GET: " + r.URL.Path)
	query := r.URL.Query()
	if !authRequest(w, query.Get("username"), query.Get("sessionid")) {
		return
	}
	name := path.Base(r.URL.Path)
	if strings.Contains(name, ".") {
		fmt.Println("download: " + name)
		file := url2path[requestUrl] + name
		if exist, _ := exists(file); !exist {
			http.NotFound(w, r)
		}
		http.ServeFile(w, r, file)
		return
	} else {
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

}

func handlePost(requestUrl string, w http.ResponseWriter, r *http.Request) (err error) {
	fmt.Println("POST: " + r.URL.Path)
	r.ParseMultipartForm(32 << 20)
	if !authRequest(w, r.Form.Get("username"), r.Form.Get("sessionid")) {
		return
	}
	file, handler, err := r.FormFile("newfile")
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
	if _, err = exists(filePath); err != nil {
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
