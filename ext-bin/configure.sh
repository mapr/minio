#/usr/bin/env bash

S3SERVER_HOME=/opt/mapr/s3server/s3server-1.0.0
WARDEN_CONF=$S3SERVER_HOME/conf/warden.s3server.conf
MINIO_BINARY=/opt/mapr/s3server/s3server-1.0.0/bin/minio
MFS_MINIO_CONFIG=s3server/s3server-1.0.0/conf/minio.json
manageSSLKeys=$MAPR_HOME/server/manageSSLKeys.sh

if [ -e "${MAPR_HOME}/server/common-ecosystem.sh" ]; then
    . ${MAPR_HOME}/server/common-ecosystem.sh
else
   echo "Failed to source common-ecosystem.sh"
   exit 0
fi

function copyWardenFile() {
    cp $WARDEN_CONF /opt/mapr/conf/conf.d
}

function tweakPermissions() {
    chown ${MAPR_USER}:${MAPR_GROUP} $MINIO_BINARY
    chown -R ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/conf

    chown ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME
    chown ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/bin
    chmod 6150 $MINIO_BINARY
    chmod 700 $S3SERVER_HOME/conf/tenants.json
}

function setupCertificate() {
    if [ ! -f $MAPR_HOME/conf/ssl_truststore.pem ]; then
        $manageSSLKeys create -N $(getClusterName) -ug $MAPR_USER:$MAPR_GROUP
    fi
    mkdir -p $S3SERVER_HOME/.minio/certs
    cp $MAPR_HOME/conf/ssl_truststore.pem $S3SERVER_HOME/.minio/certs/public.crt
}

function fixupMfsJson() {
    clustername=$(getClusterName)
    nodename=$(hostname)
    datapath="/mapr/$clustername/apps/s3/$nodename"

    if [ ! -d /mapr/$clustername/apps ]
    then
        echo "No MapRFS found on /mapr/$clustername/apps"
        exit 1
    fi

    echo "Configuring S3Server to run on $datapath"
    sed -i "s#\"fsPath\"\s*:\s*\".*\"#\"fsPath\": \"$datapath\"#g" $MFS_MINIO_CONFIG
}

setupCertificate
fixupMfsJson
tweakPermissions
copyWardenFile
