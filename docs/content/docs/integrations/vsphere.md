---
title: vSphere
main_menu: true
card: 
  name: compute
---


CDS build using VMware vSphere infrastructure to spawn each CDS Workers inside dedicated VM.

## Pre-requisites

This hatchery spawns VM which obtains IP from DHCP. So first you have to create a DHCP server on your host with NAT if you want to access to the internet. In order to create you have multiple possibilities like create your own VM with a DHCP server configured or if you are comfortable with the VMware tools you can use the [NSX system](https://www.vmware.com/products/nsx.html). This system will create DHCP gateway for you.

Also we recommend you to create a VM base that the hatchery will use to linked clone your new VM to execute your jobs. For example in our case we create different VM base with a minimal debian installed in different versions. In order to save your host resources we advice you to turn these VMs off.

## Start vSphere hatchery

Generate a token for group:

```bash
$ cdsctl token generate shared.infra persistent
expiration  persistent
created     2019-03-13 18:47:56.715104 +0100 CET
group_name  shared.infra
token       xxxxxxxxxe7x4af2d408e5xxxxxxxff2adb333fab7d05c7752xxxxxxx
```

Edit the CDS [configuration]({{< relref "/hosting/configuration.md">}}) or set the dedicated environment variables. To enable the hatchery, just set the API HTTP and GRPC URL, the token freshly generated and the vSphere variables.

This hatchery will now start worker of model 'vsphere' on vSphere infrastructure.

## Setup a worker model

See [Tutorial]({{< relref "/docs/tutorials/worker_model-vsphere.md" >}})
