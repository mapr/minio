#!/usr/bin/env bash

MINIO_DIR=/opt/mapr/s3server/s3server-1.0.0
MINIO_PID_FILE=/opt/mapr/pid/s3server.pid
MAPR_S3_CONFIG=$MINIO_DIR/conf/minio.json
MINIO_LOG_FILE=$MINIO_DIR/logs/minio.log
DEPLOYMENT_TYPE_FILE=.deployment_type

function checkSecurityScenario() {
    configMode=$(grep deploymentMode $MAPR_S3_CONFIG | sed -e "s/\s*\"deploymentMode\"\s*:\s*\"\(.*\)\",/\1/g")
    if [ -z $configMode ]
    then
        configMode="mixed"
    fi

    fsPath=$(grep fsPath $MAPR_S3_CONFIG | sed -e "s/\s*\"fsPath\"\s*:\s*\"\(.*\)\",/\1/g")

    if [ -f $fsPath/$DEPLOYMENT_TYPE_FILE ]; then
        currentMode=$(cat $fsPath/$DEPLOYMENT_TYPE_FILE)
        if [ ! $currentMode = $configMode ]; then
           echo "Warning: running on previously populated storage with different deploymentMode (previous: $currentMode)"
        fi
    fi
    mkdir -p $fsPath
    echo $configMode > $fsPath/$DEPLOYMENT_TYPE_FILE
}

if [ ! -d $MINIO_DIR ]
then
   echo "Failed to start s3server"
   exit 1
fi

case $1 in
    start)
        rm -rf $MINIO_DIR/logs
        mkdir $MINIO_DIR/logs
        echo "[$(date -R)] Running minio" >> "$MINIO_LOG_FILE"
        checkSecurityScenario >> "$MINIO_LOG_FILE" 2>&1
	    nohup $MINIO_DIR/bin/minio server dummy-arg --config-dir $MINIO_DIR/conf -M $MAPR_S3_CONFIG >> $MINIO_DIR/logs/minio.log 2>&1 & echo $! > $MINIO_PID_FILE
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
