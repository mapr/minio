#/usr/bin/env bash

S3SERVER_HOME=/opt/mapr/s3server/s3server-1.0.0
WARDEN_CONF=$S3SERVER_HOME/conf/warden.s3server.conf
MINIO_BINARY=/opt/mapr/s3server/s3server-1.0.0/bin/minio
MAPR_S3_CONFIG=s3server/s3server-1.0.0/conf/minio.json
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
    chown -R ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/conf

    chown ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME
    chown -R ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/bin
    chmod 6150 $MINIO_BINARY
    setcap "cap_setuid,cap_setgid+ei" $MINIO_BINARY
    chmod 700 $S3SERVER_HOME/conf/tenants.json
}

function setupCertificate() {
    if [ ! -f $MAPR_HOME/conf/ssl_truststore.pem ]; then
        $manageSSLKeys create -N $(getClusterName) -ug $MAPR_USER:$MAPR_GROUP
    fi
    mkdir -p $S3SERVER_HOME/conf/.minio/certs
    cp $MAPR_HOME/conf/ssl_truststore.pem $S3SERVER_HOME/conf/.minio/certs/public.crt
}

function fixupMfsJson() {
    clustername=$(getClusterName)
    nodename=$(hostname)

    if [ ! -d /mapr/$clustername/apps ]
    then
        echo "No MapRFS found on /mapr/$clustername/apps"
        exit 1
    fi

    sed -i -e "s/\${cluster}/$clustername/" -e "s/\${node}/$nodename/" $MAPR_S3_CONFIG
    fsPath=$(grep fsPath $MAPR_S3_CONFIG | sed -e "s/\s*\"fsPath\"\s*:\s*\"\(.*\)\",/\1/g")
    echo "Configuring S3Server to run on $fsPath"
}

setupCertificate
fixupMfsJson
tweakPermissions
copyWardenFile
