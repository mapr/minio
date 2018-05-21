#/usr/bin/env sh

WARDEN_CONF=/opt/mapr/s3server/s3server-1.0.0/conf/warden.s3server.conf
MINIO_BINARY=/opt/mapr/s3server/s3server-1.0.0/bin/minio

cp $WARDEN_CONF /opt/mapr/conf/conf.d
chown mapr $MINIO_BINARY
chmod 6050 $MINIO_BINARY
