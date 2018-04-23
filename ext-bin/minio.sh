#!/usr/bin/env sh

if [ -z "$(MAPR_MINIO_CONF_DIR)" ]
then
   echo ""
   exit 1
fi

case $1 in
    start)
        echo "Running minio"
        $(MAPR_SERVER_DIR)/minio --config-dir $(MAPR_MINIO_CONF_DIR) -T $(MAPR_MINIO_CONF_DIR)/tenants.json $(MAPR_MINIO_MOUNT_DIR) & echo $! > $(MAPR_MINIO_PID_FILE)
        ;;
    stop)
        if [ -f $(MAPR_MINIO_PID_FILE)]
        then
            echo "Stopping minio"
            cat $(MAPR_MINIO_PID_FILE) | xargs kill -9
        else
            echo "Minio is not running"
        fi
        ;;
    status)
        if [ -f $(MAPR_MINIO_PID_FILE)]
        then
            echo "Minio is running"
        else
            echo "Minio is not running"
        fi
