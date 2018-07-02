# Test env for multi tenancy evaluation in minio

## Description

This directory contains Vagrant environment with golang 1.9.4,
generated `~/home/vagrant/tenants.json` file and created users (tenant1, tenant2, tenant3)
which are described in `tenants.json` as tenants.

The 9000 port is forwarded to the host environment and this is where the minio listens on.

## Setting up

To bring the Vagrant VM run:
```bash
    vagrant up
    vagrant ssh
```

To build minio, execute the following inside the vagrant session:
```bash
    vagrant up
    cd gopath/src/github.com/minio/minio
    CGO_CFLAGS="-I/opt/mapr/include" CGO_LDFLAGS="-L/opt/mapr/lib -lMapRClient_c" go build
```

To run:
```bash
    cp minio /home/vagrant
    cd /home/vagrant
    sudo chown root ./minio
    sudo chmod +s minio
    mkdir -m 0777 ~/data
    LD_LIBRARY_PATH=/opt/mapr/lib ./minio server /home/vagrant/data -T /home/vagrant/tenants.json
```

Enjoy the endless fun!

## Managing tenants

Either edit the `/home/vagrant/tenants.json` or
`create_tenants()` function in `minio-test-provision.sh` and rebuild Vagrant VM.
