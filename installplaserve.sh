useradd -m steam
cd /home/steam
sudo add-apt-repository multiverse
sudo dpkg --add-architecture i386
sudo apt update
sudo apt install lib32gcc-s1 steamcmd
ln -s /usr/games/steamcmd steamcmd
./steamcmd +login anonymous +app_update 2394010 validate +quit
