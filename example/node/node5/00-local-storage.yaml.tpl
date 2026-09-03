---
apiVersion: v1alpha1
kind: UserVolumeConfig
name: storage
volumeType: disk
provisioning:
    diskSelector:
        match: disk.dev_path == "{{ .Node.Data.storageDevice }}"