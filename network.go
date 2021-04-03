package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type MsgHandle func(name string, messageBody []byte)

type NetworkManager struct {
	localPort        int64
	peerHost         string
	peerPort         int64
	sendConn         *net.UDPConn
	localAddr        *net.UDPAddr
	max              int
	messages         list.List
	messageListMutex sync.RWMutex
	messageCh        chan os.Signal
	messageHandles   map[string]MsgHandle
}

type NetMessage struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Time string `json:"time"`
}

func (this *NetworkManager) Init(max int, localPort int64, peerHost string, peerPort int64) {
	this.max = max
	this.localPort = localPort
	this.peerHost = peerHost
	this.peerPort = peerPort
	this.messageHandles = make(map[string]MsgHandle)
}

func (this *NetworkManager) RegMsgHandle(name string, handle MsgHandle) {
	this.messageHandles[name] = handle
	log.Printf("net message handle reg:%s\n", name)
}

func (this *NetworkManager) Start() {
	go func() {
		this.ReceiveService()
	}()
	go func() {
		this.SendService()
	}()
}

func (this *NetworkManager) sendMsg(message []byte) {
	this.sendConn.Write(message)
}

func (this *NetworkManager) SendAllMsg() {
	this.messageListMutex.Lock()
	if this.sendConn != nil {
		for p := this.messages.Front(); p != nil; p = p.Next() {
			v := p.Value.([]byte)
			this.sendMsg(v)
		}
	} else {
		log.Printf("udp socket invalid,msg drop.\n")
	}
	this.messages.Init()
	this.messageListMutex.Unlock()
}

func (this *NetworkManager) HandleMsg(id int, name string, data []byte) {
	log.Printf("recev msg[%d]:%s\n", id, name)
	if tempHandleFunc, ok := this.messageHandles[name]; ok {
		tempHandleFunc(name, data)
	} else {
		log.Printf("not found handle:%s\n", name)
	}
}

func (this *NetworkManager) ReceiveService() {
	localAddrStr := "0.0.0.0:" + strconv.FormatInt(this.localPort, 10)
	localAddr, err := net.ResolveUDPAddr("udp", localAddrStr)
	if err != nil {
		log.Printf("Can't resolve address: %s\n", err)
		os.Exit(1)
	}
	this.localAddr = localAddr
	conn, err := net.ListenUDP("udp", this.localAddr)
	if err != nil {
		log.Printf("Error listening:%s\n", err)
		os.Exit(1)
	}

	log.Printf("[INFO]msg recevice service start:%s\n", localAddrStr)
	defer conn.Close()
	for {
		data := make([]byte, 1024)
		dataLen, _, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Printf("failed to read UDP msg because of %s\n", err.Error())
			continue
		}
		m := new(NetMessage)
		err = json.Unmarshal(data[:dataLen], m)
		if err != nil {
			log.Printf("[ERR]msg parse to json fail:%s,err:%v\n", data, err)
			continue
		}
		this.HandleMsg(m.Id, m.Name, data[:dataLen])
	}
}

func (this *NetworkManager) SendService() {
	remoteAddr := this.peerHost + ":" + strconv.FormatInt(this.peerPort, 10)
	raddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		log.Printf("Can't resolve address: %s\n", err)
		os.Exit(1)
	}
	this.sendConn, err = net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Printf("Can't build send udp: %s\n", err)
	}
	this.messageCh = make(chan os.Signal, 1)
	signal.Notify(this.messageCh, syscall.SIGUSR1, os.Interrupt, os.Kill)
	log.Printf("[INFO]msg send service start.\n")
	for {
		s := <-this.messageCh
		if s == os.Interrupt || s == os.Kill {
			log.Println("[ERR]NetworkManager receive exit sig")
			return
		}
		if s == syscall.SIGUSR1 {
			this.SendAllMsg()
			continue
		}
		log.Println("[ERR]NetworkManager receive unkown sig", s)
	}
}
func (this *NetworkManager) SendMessage(message interface{}) {
	msg, err := json.MarshalIndent(message, "", "    ")
	if err != nil {
		log.Println("[ERR]gen msg json fail1:", err)
		return
	}

	this.messageListMutex.Lock()
	if this.messages.Len() > this.max {
		this.messages.Remove(this.messages.Front())
	}

	this.messages.PushBack(msg)
	this.messageListMutex.Unlock()
	this.messageCh <- syscall.SIGUSR1
}

type TestSendMessage struct {
	NetMessage
	T1 int64 `json:"tt"`
}

type MsgTest struct {
}

func (this *MsgTest) TestRceiverMsgProc(name string, messageBody []byte) {
	m := new(TestSendMessage)
	err := json.Unmarshal(messageBody, m)
	if err != nil {
		log.Printf("[ERR]msg parse to json fail:%s,err:%v\n", messageBody, err)
		return
	}

	fmt.Printf("receive msg:%s, TestData1:%d\n", name, m.T1)
}

func (this *MsgTest) TestNetMsg(localPort int64, peerPort int64) {
	nm := new(NetworkManager)
	nm.Init(10, localPort, "127.0.0.1", peerPort)
	nm.RegMsgHandle("TestSenderMessage", this.TestRceiverMsgProc)
	nm.Start()

	time.Sleep(time.Second * 1)
	msgBody := TestSendMessage{}
	msgBody.T1 = localPort
	msgBody.Id = 1
	msgBody.Name = "TestSenderMessage"
	msgBody.Time = getNowTime()
	nm.SendMessage(&msgBody)
	time.Sleep(time.Second * 2)
	msgBody.Id = 2
	msgBody.Time = getNowTime()
	nm.SendMessage(&msgBody)
	time.Sleep(time.Second * 2)
	msgBody.Id = 3
	msgBody.Time = getNowTime()
	nm.SendMessage(&msgBody)
	time.Sleep(time.Second * 2)
	msgBody.Id = 4
	msgBody.Time = getNowTime()
	nm.SendMessage(&msgBody)
	time.Sleep(time.Second * 2)
	msgBody.Id = 5
	msgBody.Time = getNowTime()
	nm.SendMessage(&msgBody)

}
