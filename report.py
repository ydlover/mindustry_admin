#! /usr/bin/env python
# -*- coding: utf-8 -*-
import os,re
logDir = "logs/"
#eg:[05-11-2019 | 09:07:47] [INFO] 我混一点资源 has connected.
#eg:[06-13-2019 | 22:54:09] [INFO] [8VFIBqYZMG0=] bbhghhhh has connected. 
play_conn_reg = r"\[(.*?)\s\|\s(\d\d):\d\d:\d\d\]\s+\[INFO\]\s*(.+)\s+has\sconnected\."
#eg:[06-13-2019 | 22:54:16] [INFO] [YYDr/uI9M44=] Long has disconnected.
#eg:[05-11-2019 | 09:09:06] [INFO] Long has disconnected.
play_disconn_reg = r"\[(.*?)\s\|.*?\]\s+\[INFO\]\s*(.+)\s+has\disconnected\."

class Status(object):
    def __init__(self):
        self.lastDay = ""
        self.lastHour = ""
        self.dayUsers  = []
        self.hourUsers = []
        if(not os.path.exists("./report")):
            os.mkdir("./report")
        self.dayReportCsv=open("./report/dayReport.csv","w")
        self.dayReportCsv.write("%s,%s\n"%("day","playCnt"))
        self.hourReportCsv=open("./report/hourReport.csv","w")
        self.hourReportCsv.write("%s,%s\n"%("hour","playCnt"))
    def Close(self):
        self.dayReportCsv.write("%s,%s\n"%(self.lastDay,len(self.dayUsers)))
        self.dayReportCsv.close()
        self.hourReportCsv.write('"%s",%s\n'%(self.lastHour,len(self.hourUsers)))
        self.hourReportCsv.close()
    def PlayConn(self,day,hour,userName):
        if(self.lastDay != day):
            if(self.lastDay != ""):
                self.dayReportCsv.write("%s,%s\n"%(self.lastDay,len(self.dayUsers)))
            self.lastDay = day
            self.dayUsers=[]
        self.dayUsers.append(userName)
        currHour = "%s %s:00:00"%(day,hour)
        if(self.lastHour != currHour):
            if(self.lastHour != ""):
                self.hourReportCsv.write('"%s",%s\n'%(self.lastHour,len(self.hourUsers)))
            self.lastHour = currHour
            self.hourUsers=[]
        self.hourUsers.append(userName)
    def PlayDisConn(self,userName):
        pass
    
    
def report():
    status = Status()
    files = os.listdir(logDir)
    files.sort(key= lambda x:int(x[4:-4]))
    for logFileName in files :
        for line in open(logDir+logFileName):
            objConn = re.match(play_conn_reg,line)
            if(objConn != None):
                day = objConn.group(1)
                hour = objConn.group(2)
                playUser = objConn.group(3)
                status.PlayConn(day,hour,playUser)
    status.Close()
            
if __name__ == "__main__":
    report()