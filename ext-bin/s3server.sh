#!/usr/bin/env bash

MINIO_DIR=/opt/mapr/s3server/s3server-1.0.0
MINIO_PID_FILE=/opt/mapr/pid/s3server.pid
MAPR_S3_CONFIG=$MINIO_DIR/conf/mfs.json
MINIO_LOG_FILE=$MINIO_DIR/logs/minio.log
DEPLOYMENT_TYPE_FILE=.deployment_type

function checkSecurityScenario() {
    configScenario=$(grep securityScenario $MAPR_S3_CONFIG | sed -e "s/\s*\"securityScenario\"\s*:\s*\"\(.*\)\",/\1/g")
    if [ -z $configScenario ]
    then
        configScenario="hybrid"
    fi

    fsPath=$(grep fsPath $MAPR_S3_CONFIG | sed -e "s/\s*\"fsPath\"\s*:\s*\"\(.*\)\",/\1/g")

    if [ -f $fsPath/$DEPLOYMENT_TYPE_FILE ]; then
        currentScenario=$(cat $fsPath/$DEPLOYMENT_TYPE_FILE)
        if [ ! $currentScenario = $configScenario ]; then
           echo "Warning: running on previously populated storage with different securityScenario (previous: $currentScenario)"
        fi
    fi
    mkdir -p $fsPath
    echo $configScenario > $fsPath/$DEPLOYMENT_TYPE_FILE
}

if [ ! -d $MINIO_DIR ]
then
   echo "Failed to start s3server"
   exit 1
fi

case $1 in
    start)
        echo "Running minio"
        rm -rf $MINIO_DIR/logs
        mkdir $MINIO_DIR/logs
        checkSecurityScenario 2>&1 | tee "$MINIO_LOG_FILE"
	    nohup $MINIO_DIR/bin/minio server dummy-arg --config-dir $MINIO_DIR/conf -M $MINIO_DIR/conf/mfs.json >> $MINIO_DIR/logs/minio.log 2>&1 & echo $! > $MINIO_PID_FILE
        ;;
    stop)
        if [ -f $MINIO_PID_FILE ]
        then
            echo "Stopping minio"
            cat $MINIO_PID_FILE | xargs kill -9
	    rm $MINIO_PID_FILE
        else
            echo "Minio is not running"
	    exit 1
        fi
        ;;
    status)
        if [ ! -f $MINIO_PID_FILE ]
        then
            echo "Minio is not running"
            exit 1
        fi
        if [ $(kill -0 $(cat $MINIO_PID_FILE)) ]
        then
            echo "Minio is not running"
            rm $MINIO_PID_FILE
            exit 1
        fi
        echo "Minio is running"
esac
