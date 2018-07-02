#!/usr/bin/env bash

MAPR_HOST=10.10.103.130
MAPR_CLUSTERNAME=test-mapr
MAPR_USER_UID=5000
MAPR_USER_GID=5000

function add_tenant {
    if ! grep $1 /etc/passwd
    then
        groupadd -g $4 $1
        useradd -u $3 -g $4 $1
    fi
    USER_ID=`id -u $1`
    GROUP_ID=`id -g $1`
    python3 - <<EOF
import json
f=open('$5', 'a+')
f.seek(0)
str = f.read()
data = dict()
if len(str) > 0:
    data = json.loads(str);
if 'tenants' not in data:
   data['tenants'] = []
if 'credentials' not in data:
   data['credentials'] = []
data['tenants'].append({ 'name': '$1', 'uid': '$USER_ID', 'gid': '$GROUP_ID' })
data['credentials'].append({ 'tenant': '$1', 'accessKey': '$1', 'secretKey': '$2' })
f.truncate(0)
f.write(json.dumps(data, sort_keys=True, indent=4))
EOF
}

function create_tenants {
    add_tenant mapr maprSecretKey $MAPR_USER_UID $MAPR_USER_GID /home/vagrant/tenants.json
    add_tenant tenant1 secretKey1 5001 5001 /home/vagrant/tenants.json
    add_tenant tenant2 secretKey2 5002 5002 /home/vagrant/tenants.json
    add_tenant tenant3 secretKey3 5003 5003 /home/vagrant/tenants.json
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

function setup_mapr_client {
    add-apt-repository ppa:webupd8team/java
    add-apt-repository 'deb http://package.mapr.com/releases/v6.0.0/ubuntu binary trusty' -y
    add-apt-repository 'deb http://package.mapr.com/releases/MEP/MEP-6.0/ubuntu binary trusty' -y
    apt-get update
    echo "oracle-java8-installer shared/accepted-oracle-license-v1-1 select true" | debconf-set-selections
    apt-get install oracle-java8-installer mapr-posix-client-basic mapr-client mapr-librdkafka -y --allow-unauthenticated

    chmod +x /opt/mapr/bin/fusermount
    mkdir -m 0777 /mapr
    /opt/mapr/server/configure.sh -N $MAPR_CLUSTERNAME -C $MAPR_HOST -Z $MAPR_HOST -c
    systemctl start mapr-posix-client-basic.service
}

function setup_fs {
    mkdir -pm 777 /data
}

create_tenants
apt-get install -y python3
setup_mapr_client
setup_go
setup_fs
