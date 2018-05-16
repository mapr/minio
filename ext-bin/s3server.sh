#!/usr/bin/env sh

if [ ! -d $MINIO_DIR ]
then
   echo ""
   exit 1
fi

case $1 in
    start)
        echo "Running minio"
        $MINIO_DIR/bin/minio server --config-dir $MINIO_DIR/conf -T $MINIO_DIR/conf/tenants.json $MINIO_DIR/conf & echo $! > $MINIO_PID_FILE
        ;;
    stop)
        if [ -f $MINIO_PID_FILE ]
        then
            echo "Stopping minio"
            cat $MINIO_PID_FILE | xargs kill -9
            rm $MINIO_PID_FILE
        else
            echo "Minio is not running"
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
