#!/usr/bin/env bash

function add_tenant {
    if ! grep $1 /etc/passwd
    then
        useradd $1
    fi
    USER_ID=`id -u $1`
    GROUP_ID=`id -g $1`
    python3 - <<EOF
import json
f=open('$3', 'a+')
f.seek(0)
str = f.read()
data = dict()
if len(str) > 0:
    data = json.loads(str);
data['$1'] = { 'secretKey': '$2', 'uid': '$USER_ID', 'gid': '$GROUP_ID' }
f.truncate(0)
f.write(json.dumps(data, sort_keys=True, indent=4))
EOF
}

function create_tenants {
         add_tenant tenant1 secretKey1 /home/vagrant/tenants.json
         add_tenant tenant2 secretKey2 /home/vagrant/tenants.json
         add_tenant tenant3 secretKey3 /home/vagrant/tenants.json
         cat /home/vagrant/tenants.json
}

function setup_go {
    cd /home/vagrant
    wget https://dl.google.com/go/go1.9.4.linux-amd64.tar.gz 1>&/dev/null
    tar zxf go1.9.4*tar.gz
    echo "export GOROOT=/home/vagrant/go" >> /home/vagrant/.bashrc
    echo "export GOPATH=/home/vagrant/gopath" >> /home/vagrant/.bashrc
    echo "export PATH=/home/vagrant/go/bin:$PATH" >> /home/vagrant/.bashrc
}

function setup_fs {
    mkdir -pm 777 /data
}

apt-get install -y python3
setup_go
setup_fs
create_tenants
