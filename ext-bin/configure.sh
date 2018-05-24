#/usr/bin/env bash

S3SERVER_HOME=/opt/mapr/s3server/s3server-1.0.0
WARDEN_CONF=$S3SERVER_HOME/conf/warden.s3server.conf
MINIO_BINARY=/opt/mapr/s3server/s3server-1.0.0/bin/minio
MFS_MINIO_CONFIG=s3server/s3server-1.0.0/conf/mfs.json

function copyWardenFile() {
    cp $WARDEN_CONF /opt/mapr/conf/conf.d
}

function tweakPermissions() {
    chown root:${MAPR_GROUP} $MINIO_BINARY
    chown -R ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/conf

    chown ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME
    chown ${MAPR_USER}:${MAPR_GROUP} $S3SERVER_HOME/bin
    chmod 6050 $MINIO_BINARY
}

function fixupMfsJson() {
    clustername=$(cat /opt/mapr/conf/mapr-clusters.conf | cut -d" " -f1)
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

fixupMfsJson
tweakPermissions
copyWardenFile
