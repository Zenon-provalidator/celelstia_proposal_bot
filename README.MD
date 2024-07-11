## 1. Install redis
```
sudo apt-get install redis-server -y
```

## 2. Install go
```
sudo rm -rf /usr/local/go

curl https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz | sudo tar -C/usr/local -zxvf -

# Update environment variables to include go
cat <<'EOF' >>$HOME/.profile
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export GO111MODULE=on
export GOBIN=$GOPATH/bin
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
EOF
source $HOME/.profile
```

## 2. Go mod install
```
go get github.com/go-redis/redis/v9
go get github.com/go-telegram-bot-api/telegram-bot-api
go get "gopkg.in/yaml.v3"
```

## 3. Variable setting
cp config-yaml config.yaml
```
api:
  url: "celestia-lcd/cosmos/gov/v1beta1/proposals"

explorer:
  url: "https://www.mintscan.io/celestia-testnet/proposals/"
  
redis:
  addr: "localhost:6379"
  password: "password"
  db_index: 1

telegram:
  bot_token: "telegram bot token"
  chat_id: -100

ticker:
  interval: 1m
```

## 4. Service
```
echo "[Unit]
Description=celestia_proposal_bot
After=network-online.target

[Service]
User=$USER
ExecStart=go run main.go

Restart=on-failure
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
" >celestia_proposal_bot.service

sudo mv celestia_proposal_bot.service /etc/systemd/system/celestia_proposal_bot.service
sudo -S systemctl daemon-reload
sudo systemctl enable celestia_proposal_bot
sudo systemctl start celestia_proposal_bot
journalctl -fu celestia_proposal_bot
sudo systemctl stop celestia_proposal_bot
```