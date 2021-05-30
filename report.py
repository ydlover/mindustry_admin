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
#eg:2019/06/23 01:35:00 tenMinTask trig[48.312°C].
minTask_reg = r"(.*?)\s(\d\d):\d\d:\d\d\s+tenMinTask\strig\[(.*)°C\]\."

class Status(object):
    def __init__(self, isReportAdmin):
        self.lastDay = ""
        self.lastHour = ""
        self.lastAdminDay = ""
        self.lastAdminHour = ""
        self.dayUsers  = []
        self.hourUsers = []
        self.totalTemp = 0
        self.totalHourTemp = 0
        self.tempCnt =0
        self.tempHourCnt =0
        if(not os.path.exists("./report")):
            os.mkdir("./report")
        self.dayReportCsv=open("./report/dayReport.csv","w")
        self.dayReportCsv.write("%s,%s\n"%("day","playCnt"))
        self.hourReportCsv=open("./report/hourReport.csv","w")
        self.hourReportCsv.write("%s,%s\n"%("hour","playCnt"))
        self.isReportAdmin = isReportAdmin
        if self.isReportAdmin :
            self.dayAdminReportCsv=open("./report/dayAdminReport.csv","w")
            self.dayAdminReportCsv.write("%s,%s\n"%("day","temp"))
            self.hourAdminReportCsv=open("./report/hourAdminReport.csv","w")
            self.hourAdminReportCsv.write("%s,%s\n"%("hour","temp"))
    def Close(self):
        self.dayReportCsv.write("%s,%s\n"%(self.lastDay,len(self.dayUsers)))
        self.dayReportCsv.close()
        self.hourReportCsv.write('"%s",%s\n'%(self.lastHour,len(self.hourUsers)))
        self.hourReportCsv.close()
        if self.isReportAdmin :
            self.dayAdminReportCsv.write("%s,%f\n"%(self.lastAdminDay,self.totalTemp/self.tempCnt))
            self.dayAdminReportCsv.close()
            self.hourAdminReportCsv.write("%s,%f\n"%(self.lastAdminHour,self.totalHourTemp/self.tempHourCnt))
            self.hourAdminReportCsv.close()
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
    
    def TempCnt(self,day,hour,temp):
        if(self.lastAdminDay != day):
            if(self.lastAdminDay != ""):
                self.dayAdminReportCsv.write("%s,%f\n"%(self.lastAdminDay,self.totalTemp/self.tempCnt))
            self.lastAdminDay = day
            self.totalTemp = 0
            self.tempCnt = 0
        self.totalTemp += temp
        self.tempCnt += 1

        currHour = "%s %s:00:00"%(day,hour)
        if(self.lastAdminHour != currHour):
            if(self.lastAdminHour != ""):
                self.hourAdminReportCsv.write("%s,%f\n"%(self.lastAdminHour,self.totalHourTemp/self.tempHourCnt))
            self.lastAdminHour = currHour
            self.totalHourTemp = 0
            self.tempHourCnt = 0

        self.totalHourTemp += temp
        self.tempHourCnt += 1
    
def report(isReportAdmin):
    status = Status(isReportAdmin)
    files = os.listdir(logDir)
    files.remove("admin.log")
    files.sort(key= lambda x:int(x[4:-4]))
    for logFileName in files:
        for line in open(logDir+logFileName):
            objConn = re.match(play_conn_reg,line)
            if(objConn != None):
                day = objConn.group(1)
                hour = objConn.group(2)
                playUser = objConn.group(3)
                status.PlayConn(day,hour,playUser)

    if isReportAdmin :
        for line in open(logDir+"admin.log"):
             objTemp = re.match(minTask_reg,line)
             if(objTemp != None):
                day = objTemp.group(1)
                hour = objTemp.group(2)
                temp = float(objTemp.group(3))
                status.TempCnt(day,hour,temp)
             
    status.Close()
import sys
if __name__ == "__main__":
    isReportAdmin = False
    if len(sys.argv)>1 and sys.argv[1]=="1":
        isReportAdmin = True
    report(isReportAdmin)
