#!/usr/bin/env sh

MINIO_DIR=/opt/mapr/s3server/s3server-1.0.0
MINIO_PID_FILE=/opt/mapr/pid/s3server.pid

if [ ! -d $MINIO_DIR ]
then
   echo "Failed to start s3server"
   exit 1
fi

echo $1

case $1 in
    start)
        echo "Running minio"
	    nohup $MINIO_DIR/bin/minio server dummy-arg --config-dir $MINIO_DIR/conf -M $MINIO_DIR/config/mfs.json  > /dev/null 2>&1 & echo $! > $MINIO_PID_FILE
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
        if [ -f $MINIO_PID_FILE ]
        then
            echo "Minio is running"
		exit 0
        else
            echo "Minio is not running"
		exit 1
        fi
esac
