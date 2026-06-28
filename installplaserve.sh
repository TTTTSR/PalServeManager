#依赖准备
useradd -m steam
cd /home/steam
sudo add-apt-repository multiverse
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install lib32gcc-s1 steamcmd
ln -s /usr/games/steamcmd steamcmd
./steamcmd +login anonymous +app_update 2394010 validate +quit

#拉取脚本和程序
mkdir -p /opt/palworld-manager/logs
curl -fsSL https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palservemanage.sh -o /opt/palworld-manager/palservemanage.sh
curl -fsSL https://raw.githubusercontent.com/TTTTSR/PalServeManager/main/palworld-manager-linux -o /opt/palworld-manager/palworld-manager-linux
chmod +x /opt/palworld-manager/palservemanage.sh /opt/palworld-manager/palworld-manager-linux
chown -R steam:steam /opt/palworld-manager