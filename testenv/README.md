# Test env for multi tenancy evaluation in minio

## Description

This directory contains Vagrant environment with golang 1.9.4,
generated `~/home/vagrant/tenants.json` file and created users (tenant1, tenant2, tenant3)
which are described in `tenants.json` as tenants.

The 9000 port is forwarded to the host environment and this is where the minio listens on.

## Setting up

To brirg the Vagrant VM run:
```bash
    vagrant up
    vagrant ssh
```

To run minio, execute the following inside the vagrant session:

```bash
    vagrant up
    # Run everything as root
    sudo su
    mkdir -m 0777 ~/data
    cd gopath/src/github.com/minio/mino
    # Build minio
    go build
    ./minio server ~/data -T /home/vagrant/tenants.json
```

Enjoy the endless fun!

## Managing tenants

Either edit the `/home/vagrant/tenants.json` or
`create_tenants()` function in `minio-test-provision.sh` and rebuild Vagrant VM.
