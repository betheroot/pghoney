set -e
set -x

if [ $# -ne 2 ]
    then
        echo "Wrong number of arguments supplied."
        echo "Usage: $0 <server_url> <deploy_key>."
        exit 1
fi

server_url=$1
deploy_key=$2

wget $server_url/static/registration.txt -O registration.sh
chmod 755 registration.sh
# Note: this will export the HPF_* variables
. ./registration.sh $server_url $deploy_key "pghoney"

apt-get update
apt-get -y install git supervisor

# Install golang
curl -O https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
tar -xvf go1.8.linux-amd64.tar.gz
mv go /usr/local
echo "export PATH=$PATH:/usr/local/go/bin" > /etc/profile.d/go-path.sh

# Get the pghoney source
cd /opt
git clone git@github.com:ajvb/pghoney.git
cd pghoney

export GOPATH=/opt/pghoney
/usr/local/go/bin/go get || true
/usr/local/go/bin/go build

cat > pghoney.conf<<EOF
{
  "port":5432,
  "address":"0.0.0.0",
  "pgUsers":["postgres"],
  "debug":false,
  "cleartext":false,
  "server_timeout":10,
  "hpfeedsConfig":{
    "host": "$HPF_HOST",
    "port": $HPF_PORT,
    "ident": "$HPF_IDENT",
    "secret": "$HPF_SECRET",
    "channel":"pghoney.events",
    "enabled":true
  }
}

EOF

# Config for supervisor.
cat > /etc/supervisor/conf.d/pghoney.conf <<EOF
[program:pghoney]
command=/opt/pghoney/pghoney
directory=/opt/pghoney
stdout_logfile=/opt/pghoney/pghoney.out
stderr_logfile=/opt/pghoney/pghoney.err
autostart=true
autorestart=true
redirect_stderr=true
stopsignal=QUIT
EOF

supervisorctl update

