ps -ef | grep mindustry_admin | grep -v grep | awk '{print $2}' | xargs kill -9
ps -ef | grep server-release.jar | grep -v grep | awk '{print $2}' | xargs kill -9
./mindustry_admin
