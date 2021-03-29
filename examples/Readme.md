#Quick start and examples
##Minio Client
In Objectstore 2.1.0 MC is located in _/opt/mapr/objectstore-client/objectstore-client-2.1.0/util/mc_

To configure mc run:

```
mc alias set ALIAS http(s)://HOSTNAME/IP_ADDRESS:PORT ADMIN_ACCESS_KEY ADMIN_SECRET_KEY
```

For example:

```
mc alias set myminio http://localhost:9000 minioadmin minioadmin
```

New user without UID and GID (UID and GID of Objectstore's process will be used) can be added with command: 

```
mc admin user add ALIAS/ USERNAME PASSWORD
```

For example:

```
mc admin user add myminio/ test qwerty78
```

New user with UID and GID can be added like: 

```
mc admin user add ALIAS/ USERNAME PASSWORD UID GID
```

For example:

```
mc admin user add myminio/ test qwerty78 1000 1000
```

More information about work with MC can be found in [MinIO Admin Complete Guide](https://github.com/minio/mc/blob/RELEASE.2021-03-23T05-46-11Z/docs/minio-admin-complete-guide.md)

##Examples with built in users
In examples default admin user _minioadmin_ is using.
Examples show:
* listing buckets
* creating bucket
* deleting bucket
* checking if bucket exists  
* listing files
* uploading file
* deleting file
* checking if file exists
### Java example
Java example can be found in [Objectstore.java](./java/src/main/java/org/example/objectstore/Objectstore.java).

Operations are in [Operations.java](./java/src/main/java/org/example/objectstore/Operations.java)

This example has dependencies for:

```
com.amazonaws:aws-java-sdk-s3:1.11.754
```
### Python example
Python example can be found in [objectstore.py](./python/objectstore.py).

Operations are in [operations.py](./python/operations.py).

This example has dependencies for:

```
boto3                   1.15.13
botocore                1.16.26
```
### Hadoop example
To work with Hadoop it is necessary to provide accessKey, secretKey, host and port.

For example:

```
-Dfs.s3a.access.key=ACCESS_KEY -Dfs.s3a.secret.key=PASSWORD -Dfs.s3a.endpoint=http(s)://HOST:PORT -Dfs.s3a.path.style.access=true -Dfs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem
```

### Spark example
Spark uses Hadoop libraries to work with S3, so it is also necessary to provide accessKey, secretKey, host and port.

For example:

```
--conf spark.hadoop.fs.s3a.access.key=ACCESS_KEY --conf spark.hadoop.fs.s3a.secret.key=PASSWORD --conf spark.hadoop.fs.s3a.endpoint=http(s)://HOST:PORT --conf spark.hadoop.fs.s3a.path.style.access=true --conf spark.hadoop.fs.s3a.impl=org.apache.hadoop.fs.s3a.S3AFileSystem
```

## LDAP/AD Integration
Objectstore integration uses MinIO LDAP STS, full documentation can be found here [AssumeRoleWithLDAPIdentity](https://github.com/minio/minio/blob/RELEASE.2021-03-17T02-33-02Z/docs/sts/ldap.md)

MinIO LDAP STS allows generating temporary credentials (accessKey, secretKey, sessionToken) that can be used for work with S3 endpoint via a client, which has support of sessionToken
### LDAP test environment
If you don't have LDAP/AD environment, you can use test environment from [docker-compose.yml](./ldap-env/docker-compose.yml).

It creates user `cn=admin,dc=mapr,dc=local` with password `abc@123`
### Objectstore configuration
To work with LDAP/AD it is necessary to configure LDAP integration for Objectstore, for test environment it can be: 

```json
"ldap": {
"serverAddr": "localhost:389",
"usernameFormat": "cn=%s,dc=mapr,dc=local",
"userDNSearchBaseDN": "",
"userDNSearchFilter": "(cn=%s)",
"groupSearchFilter": "(&(objectclass=group)(member=%s))",
"groupSearchBaseDn": "dc=mapr,dc=local",
"lookUpBindDN": "",
"lookUpBindPassword": "",
"stsExpiry": "60h",
"tlsSkipVerify": "on",
"serverStartTls": "",
"seve``rInsecure": "on"
}
```

Then you need to grant policy for users/groups

For example:

```
mc admin policy set myminio readwrite user="cn=admin,dc=mapr,dc=local"
```

### Cli example

To get credentials you have to make POST request.

In this request your special symbols in the password should be replaced with %HEX_VALUE of your [ASCII](https://www.sciencebuddies.org/science-fair-projects/references/ascii-table) symbol.

For password `abc@123` it would be `abc%40123`

For example:

```
curl -X POST "http://127.0.0.1:9000?Action=AssumeRoleWithLDAPIdentity&LDAPUsername=admin&LDAPPassword=abc%40123&Version=2011-06-15" | xmllint --format -
```

You will receive response:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<AssumeRoleWithLDAPIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithLDAPIdentityResult>
<Credentials>
<AccessKeyId>N71HK1WE34R2D7F9FDVP</AccessKeyId>
<SecretAccessKey>NmrkNOXA696CrblWU+eUn0NBwUv+4oUs2u8noJAA</SecretAccessKey>
<UID/>
<GID/>
<Expiration>2021-04-02T23:24:59Z</Expiration>
<SessionToken>eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NLZXkiOiJONzFISzFXRTM0UjJEN0Y5RkRWUCIsImV4cCI6MTYxNzQwNTg5OSwibGRhcFVzZXIiOiJjbj1hZG1pbixkYz1tYXByLGRjPWxvY2FsIn0.KY0i3DyOM-IKXi_BHADxZksC8x2PDqjDNBQVIfG-uxBKiJdHrRCnwXUy0GSGX4Q_XXvhAO4aKj5IIauDc_UceQ</SessionToken>
</Credentials>
</AssumeRoleWithLDAPIdentityResult>
<ResponseMetadata>
<RequestId>167169A551AF88C8</RequestId>
</ResponseMetadata>
</AssumeRoleWithLDAPIdentityResponse>
```

### Java example
Java example can be found in [ObjectstoreLdap.java](./java/src/main/java/org/example/objectstore/ObjectstoreLdap.java).

Operations are in [Operations.java](./java/src/main/java/org/example/objectstore/Operations.java)

This example has dependencies for:

```
com.amazonaws:aws-java-sdk-s3:1.11.754
com.squareup.okhttp3:okhttp:4.2.2
com.fasterxml.jackson.dataformat:jackson-dataformat-xml:2.8.5
```
### Python example
Python example can be found in [objectstore_ldap.py](./python/objectstore_ldap.py).

Operations are in [operations.py](./python/operations.py).

This example has dependencies for:
```
boto3                   1.15.13
botocore                1.16.26
requests                2.24.0
xmltodict               0.12.0
urllib3                 1.24.3
```

### Hadoop and Spark
MapR's implementation of Hadoop and Spark does not support sessionToken, so they can't work with Objecstore wit LDAP/AD