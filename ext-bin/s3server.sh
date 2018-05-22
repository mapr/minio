#!/usr/bin/env sh

MINIO_DIR=/opt/mapr/s3server/s3server-1.0.0
MINIO_PID_FILE=/opt/mapr/pid/s3server.pid

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
	    nohup $MINIO_DIR/bin/minio server dummy-arg --config-dir $MINIO_DIR/conf -M $MINIO_DIR/conf/mfs.json  > $MINIO_DIR/logs/minio.log 2>&1 & echo $! > $MINIO_PID_FILE
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
        if [ ! kill -0 $(cat $MINIO_PID_FILE) > /dev/null 2>&1 ]
        then
            echo "Minio is not running"
            rm $MINIO_PID_FILE
            exit 1
        fi
        echo "Minio is running"
esac
