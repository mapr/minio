#!/usr/bin/env bash

INSTALL_DIR=${MAPR_HOME:=/opt/mapr}
OBJECTSERVER_HOME=$INSTALL_DIR/objectserver/objectserver-1.0.0
WARDEN_CONF=$OBJECTSERVER_HOME/conf/warden.objectserver.conf
MINIO_BINARY=$OBJECTSERVER_HOME/bin/minio
MAPR_S3_CONFIG=$OBJECTSERVER_HOME/conf/minio.json
manageSSLKeys=$MAPR_HOME/server/manageSSLKeys.sh
sslKeyStore=${INSTALL_DIR}/conf/ssl_keystore
storePass=mapr123
storeFormat=JKS
storeFormatPKCS12=pkcs12
isSecure=`cat /opt/mapr/conf/mapr-clusters.conf | sed 's/.*\(secure=\)\(true\|false\).*/\2/'`
isClient=false

while [ ${#} -gt 0 ]; do
  case "$1" in
    -c)
      isClient=true
      shift 1;;
    -u|--user)
      MAPR_USER=`id -u $2`
      shift 2;;
    -g|--group)
      MAPR_GROUP=`id -g $2`
      shift 2;;
    -p|--path)
      optionalFsPath=$2
      if [ ! -d "$optionalFsPath" ]; then
      echo "Path does not exist."
      echo "Please specify path for file system"
      exit 1
      fi
      shift 2;;
    *)
      shift 1
  esac
done



if $isClient ; then
    if [ "${MAPR_USER}"x == "x" ] ; then
    echo "Please specify user name"
    errExit=true
    fi

    if [ "${MAPR_GROUP}"x == "x" ] ; then
    echo "Please specify group name"
    errExit=true
    fi

    if [ "${optionalFsPath}"x == "x" ] ; then
    echo "Please specify path for file system"
    errExit=true
    fi

    if [ "$errExit"x == "truex" ] ; then
    exit 1
    fi

    clustername=`cat /opt/mapr/conf/mapr-clusters.conf | sed 's/\(.*\)\( secure\).*/\1/'`

else
    if [ -e "${MAPR_HOME}/server/common-ecosystem.sh" ]; then
        . ${MAPR_HOME}/server/common-ecosystem.sh
    else
       echo "Failed to source common-ecosystem.sh"
       exit 0
    fi

    MAPR_USER=${MAPR_USER}
    MAPR_GROUP=${MAPR_GROUP}
    clustername=$(getClusterName)
    nodename=$(hostname)
fi

if [ "$JAVA_HOME"x = "x" ]; then
  KEYTOOL=`which keytool`
else
  KEYTOOL=$JAVA_HOME/bin/keytool
fi

# Check if keytool is actually valid and exists
if [ ! -e "${KEYTOOL:-}" ]; then
    echo "The keytool in \"${KEYTOOL}\" does not exist."
    echo "Keytool not found or JAVA_HOME not set properly. Please install keytool or set JAVA_HOME properly."
    exit 1
fi

function copyWardenFile() {
    cp $WARDEN_CONF /opt/mapr/conf/conf.d
}

function tweakPermissions() {
    chown -R ${MAPR_USER}:${MAPR_GROUP} $OBJECTSERVER_HOME/conf
    chown ${MAPR_USER}:${MAPR_GROUP} $OBJECTSERVER_HOME
    chown -R ${MAPR_USER}:${MAPR_GROUP} $OBJECTSERVER_HOME/bin
    chmod 6150 $MINIO_BINARY
    setcap "cap_setuid,cap_setgid+eip" $MINIO_BINARY
    chmod 700 $OBJECTSERVER_HOME/conf/tenants.json
}

function extractPemKey() {
  from=$1
  to=$2
  base_from=$(basename $from)
  sslKeyStoreP12="/tmp/${base_from}.p12"
  if [ -f "$sslKeyStoreP12" ]; then
    rm "$sslKeyStoreP12"
  fi
  if [ ! -f "$from" ]; then
    echo "Source key store not found: $from"
    return 1
  fi
  if [ -f "$to" ]; then
    echo "Destination key already exists: $to"
    return 1
  fi
  $KEYTOOL -importkeystore -srckeystore $from -destkeystore $sslKeyStoreP12 \
            -srcstorepass $storePass -deststorepass $storePass\
            -srcstoretype $storeFormat -deststoretype $storeFormatPKCS12 -noprompt $VERBOSE
  if [ $? -ne 0 ]; then
	echo "Keytool command to create P12 trust store failed"
    return 1
  fi
  openssl $storeFormatPKCS12 -nodes -nocerts -in $sslKeyStoreP12 -out $to -passin pass:$storePass
  if [ $? -ne 0 ]; then
	echo "openssl command to create PEM trust store failed"
  fi
  rm $sslKeyStoreP12
}

function setupCertificate() {
    if [ ! -f $MAPR_HOME/conf/ssl_truststore.pem ]; then
        if [ ! -f $MAPR_HOME/conf/ssl_truststore ]; then
            $manageSSLKeys create -N $clustername -ug $MAPR_USER:$MAPR_GROUP
        else
            $manageSSLKeys convert -N $clustername $MAPR_HOME/conf/ssl_truststore $MAPR_HOME/conf/ssl_truststore.pem
        fi
    fi
    mkdir -p $OBJECTSERVER_HOME/conf/certs
    cp $MAPR_HOME/conf/ssl_truststore.pem $OBJECTSERVER_HOME/conf/certs/public.crt
    extractPemKey $MAPR_HOME/conf/ssl_keystore $OBJECTSERVER_HOME/conf/certs/private.key
}

function fixupMfsJson() {
    if [ "$optionalFsPath"x == "x" ]; then
        sed -i -e "s/\${cluster}/$clustername/" -e "s/\${node}/$nodename/" $MAPR_S3_CONFIG
    else
        sed -i "s#\(\"fsPath\": \"\)\(.*\)\(\",\)#\1$optionalFsPath\3#" $MAPR_S3_CONFIG
    fi
    fsPath=$(grep fsPath $MAPR_S3_CONFIG | sed -e "s/\s*\"fsPath\"\s*:\s*\"\(.*\)\",/\1/g")
    echo "Configuring Objectserver to run on $fsPath"
}

if [ "x$isSecure" == "xtrue" ]; then
setupCertificate
fi

fixupMfsJson
tweakPermissions
if [ "x$isClient" == "xfalse" ] ; then
copyWardenFile
fi