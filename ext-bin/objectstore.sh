#!/usr/bin/env bash

MAPR_HOME=/opt/mapr
MINIO_DIR=$MAPR_HOME/objectstore-client/objectstore-client-2.0.0
MINIO_PID_FILE=$MAPR_HOME/pid/objectstore.pid
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
   echo "Failed to start objectstore"
   exit 1
fi

case $1 in
    start)
        if [ ! -d $MINIO_DIR/logs ]
        then
            mkdir $MINIO_DIR/logs
        fi

        #Setting port
        if [ -f "$MAPR_HOME/conf/conf.d/warden.objectstore.conf" ]; then
        port=$(cat $MAPR_HOME/conf/conf.d/warden.objectstore.conf | grep 'service.port=' | sed  's/\(service.port=\)//')
        sed -i  "s/\(.*\"\)\([0-9]\{1,4\}\)\(\"\)/\1$port\3/" $MAPR_S3_CONFIG
        else
        port=$(cat $MAPR_S3_CONFIG | grep 'port' | sed  's/.*\"\([0-9]\{1,5\}\)\".*/\1/')
        fi

        mountPath=$(cat $MAPR_S3_CONFIG | grep 'fsPath' | sed -e "s/\s*\"fsPath\"\s*:\s*\"\(.*\)\",/\1/g")

        echo "[$(date -R)] Minio pre-flight check" >> "$MINIO_LOG_FILE"
        checkSecurityScenario >> "$MINIO_LOG_FILE" 2>&1
        $MINIO_DIR/bin/minio server $mountPath -M $MAPR_S3_CONFIG --address :$port --check-config
        if [ $? -ne 0 ]
        then
            echo "Minio pre-flight check failed"
            exit 1
        fi
        echo "[$(date -R)] Running minio" >> "$MINIO_LOG_FILE"
            nohup $MINIO_DIR/bin/minio server $mountPath -M $MAPR_S3_CONFIG --address :$port > /dev/null 2>&1 & echo $! > $MINIO_PID_FILE

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
